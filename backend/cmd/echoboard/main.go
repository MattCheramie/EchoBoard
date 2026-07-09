// Command echoboard is the EchoBoard server and CLI entry point.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MattCheramie/echoboard/internal/account"
	"github.com/MattCheramie/echoboard/internal/api"
	"github.com/MattCheramie/echoboard/internal/auth"
	"github.com/MattCheramie/echoboard/internal/config"
	"github.com/MattCheramie/echoboard/internal/db"
	"github.com/MattCheramie/echoboard/internal/integrations"
	"github.com/MattCheramie/echoboard/internal/invite"
	"github.com/MattCheramie/echoboard/internal/user"
	"golang.org/x/term"
)

func main() {
	setup := flag.Bool("setup", false, "run first-time admin bootstrap and exit")
	addr := flag.String("addr", "", "override listen address (e.g. :8080)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(*setup, *addr, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(setup bool, addr string, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	database, err := db.Open(cfg)
	if err != nil {
		return err
	}
	defer database.Close()

	ctx := context.Background()
	if err := database.Migrate(ctx); err != nil {
		return err
	}

	users := user.NewRepository(database)
	invites := invite.NewRepository(database)
	accounts := account.NewService(users, invites)
	sessions := auth.NewSessionStore(database, 0)
	authr := auth.NewAuthenticator(sessions, users, cfg.IsProduction())

	if setup {
		return runSetup(ctx, accounts)
	}

	// The secrets vault encrypts integration tokens at rest. Warn loudly if the
	// key is ephemeral, since those secrets would not survive a restart.
	vault, err := auth.NewVault(cfg.SecretKey)
	if err != nil {
		return err
	}
	if vault.Ephemeral() {
		log.Warn("SECRET_KEY not set: integration secrets use a dev-only ephemeral key")
	}

	// Integration framework: register the built-in sandbox adapter; real platform
	// adapters (Meta, TikTok, …) register here in Tier 4 PRs 4.2–4.5.
	registry := integrations.NewRegistry()
	registry.Register(integrations.NewSandbox())
	integSvc := integrations.NewService(integrations.Options{
		Registry: registry,
		Repo:     integrations.NewRepository(database, vault),
		Vault:    vault,
		BaseURL:  cfg.PublicAPIBaseURL,
	})

	server := api.New(cfg, accounts, users, sessions, authr, integSvc, log)
	listen := addr
	if listen == "" {
		listen = fmt.Sprintf(":%d", cfg.Port)
	}
	return serve(listen, server.Handler(), log)
}

// serve runs the HTTP server until an interrupt, then shuts down gracefully.
func serve(addr string, handler http.Handler, log *slog.Logger) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-stop:
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// runSetup performs the interactive first-run admin bootstrap.
func runSetup(ctx context.Context, accounts *account.Service) error {
	fmt.Println("EchoBoard first-run setup — create the master admin account.")
	reader := bufio.NewReader(os.Stdin)

	email, err := prompt(reader, "Admin email: ")
	if err != nil {
		return err
	}
	name, err := prompt(reader, "Admin name: ")
	if err != nil {
		return err
	}
	password, err := promptPassword(reader, "Admin password: ")
	if err != nil {
		return err
	}

	admin, err := accounts.CreateFirstAdmin(ctx, email, name, password)
	if errors.Is(err, account.ErrAlreadyBootstrapped) {
		return fmt.Errorf("setup: an account already exists; setup can only run on a fresh instance")
	}
	if err != nil {
		return err
	}
	fmt.Printf("\nCreated admin %s (%s). You can now start the server.\n", admin.Name, admin.Email)
	return nil
}

func prompt(r *bufio.Reader, label string) (string, error) {
	fmt.Print(label)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads a password without echo when stdin is a terminal, and
// falls back to a plain line read otherwise (piped input, CI, automation).
func promptPassword(r *bufio.Reader, label string) (string, error) {
	fmt.Print(label)
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
