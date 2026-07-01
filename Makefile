# EchoBoard developer tasks.
# NOTE (Tier 0): the project is skeleton-only. Targets describe the intended
# workflow; several are no-ops until real code lands in Tier 1+.

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: dev
dev: ## Run backend and frontend in dev mode (Tier 1+)
	@echo "dev: not wired yet — see ROADMAP.md Tier 1"

.PHONY: build
build: build-frontend build-backend ## Build the single binary (embeds frontend)

.PHONY: build-frontend
build-frontend: ## Build the SvelteKit frontend (Tier 1+)
	@echo "build-frontend: not wired yet — see ROADMAP.md Tier 1"

.PHONY: build-backend
build-backend: ## Build the Go backend binary
	cd backend && go build -o dist/echoboard ./cmd/echoboard

.PHONY: setup
setup: build-backend ## Run first-run admin bootstrap
	cd backend && ./dist/echoboard --setup

.PHONY: embed
embed: ## Copy frontend build output into backend/internal/web/build for embedding (Tier 6)
	@echo "embed: not wired yet — see ROADMAP.md Tier 6 (PR 6.1)"

.PHONY: test
test: ## Run backend tests (frontend tests land in Tier 1 PR 1.5)
	cd backend && go test ./...

.PHONY: lint
lint: ## Run go vet (golangci-lint wired later)
	cd backend && go vet ./...
