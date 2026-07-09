# EchoBoard server — multi-stage Docker build.
#
#   Stage 1 (builder)  pure-Go build (CGO_ENABLED=0). modernc SQLite + pgx
#                      mean no C toolchain or system libraries are required.
#   Stage 2 (runtime)  carries only the server binary on a slim base.
#
# NOTE: EchoBoard's single binary will embed the built Svelte frontend
# (ROADMAP Tier 6, built with `-tags embed`). Until that lands, this image
# builds the BACKEND ONLY — the server runs and serves the REST/WebSocket API
# plus a placeholder web page. Add a Node/Vite frontend build stage and the
# `-tags embed` flag here once the embed is wired.

FROM golang:1.25-bookworm AS builder
WORKDIR /src

# Cache deps before copying the rest of the source. go.mod/go.sum live under
# backend/ in this monorepo.
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

ARG VERSION=docker
ENV CGO_ENABLED=0
RUN go build -trimpath \
        -ldflags "-s -w -X github.com/MattCheramie/echoboard/internal/api.Version=${VERSION}" \
        -o /out/echoboard ./cmd/echoboard

# ---------------------------------------------------------------

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates wget \
 && rm -rf /var/lib/apt/lists/*

# Non-root user; /data holds the SQLite database (mount a volume there).
RUN useradd --system --create-home --shell /usr/sbin/nologin echo \
 && mkdir -p /data && chown echo:echo /data
USER echo
WORKDIR /data

COPY --from=builder /out/echoboard /usr/local/bin/echoboard

# Default listen port. Store the SQLite DB on the mounted /data volume; set
# DB_DRIVER=postgres + DATABASE_URL=postgres://... for a Postgres deployment.
ENV APP_ENV=production \
    DB_DRIVER=sqlite \
    DATABASE_URL=/data/echoboard.db
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget --quiet --spider http://127.0.0.1:8080/health || exit 1

ENTRYPOINT ["/usr/local/bin/echoboard"]
