# syntax=docker/dockerfile:1.7

ARG NODE_VERSION=22
ARG GO_VERSION=1.26

# ---------- Stage 1: build frontend (R3F + HUD) ----------
FROM oven/bun:1-alpine AS web
WORKDIR /web
COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile || bun install
COPY web/ ./
RUN bun run build   # output: /web/dist

# ---------- Stage 2: build Go server (static, CGO off) ----------
# Memakai modernc.org/sqlite (pure Go) agar tidak butuh CGO/libc,
# sehingga cross-compile amd64 + arm64 cepat dan binary benar-benar statis.
FROM golang:${GO_VERSION}-alpine AS server
WORKDIR /src
# Tunggu stage web selesai agar build berjalan sekuensial (mencegah VPS freeze/OOM)
COPY --from=web /web/package.json /tmp/web_done
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd/ cmd/
COPY internal/ internal/
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/dioramaops ./cmd/dioramaops
# Siapkan folder dengan owner non-root (distroless tidak punya shell)
RUN mkdir -p /out/data /out/logos && chown -R 65532:65532 /out/data /out/logos

# ---------- Stage 3: runtime (distroless, non-root) ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=server /out/dioramaops /app/dioramaops
COPY --from=web    /web/dist    /app/public
COPY --from=server --chown=65532:65532 /out/data  /app/data
COPY --from=server --chown=65532:65532 /out/logos /app/public/logos

ENV PORT=7437 \
    DATA_DIR=/app/data \
    STATIC_DIR=/app/public \
    TZ=Asia/Jakarta

USER 65532:65532
EXPOSE 7437
VOLUME ["/app/data"]

# Tanpa curl/wget di distroless: healthcheck lewat subcommand binary sendiri
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/dioramaops", "healthcheck"]

ENTRYPOINT ["/app/dioramaops"]
CMD ["serve"]
