# syntax=docker/dockerfile:1

# Build stage
# Keep this Go version in sync with backend/go.mod (the `go` directive) and the
# CI workflows (which read go-version-file: backend/go.mod). Docker FROM cannot
# read go.mod, so this one is updated by hand.
FROM golang:1.26.4 AS build
WORKDIR /src

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static build: CGO disabled (modernc.org/sqlite is pure Go)
ENV CGO_ENABLED=0
RUN go build -o /out/server -ldflags="-s -w" ./cmd/server/

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
USER nonroot:nonroot
# Distroless has no shell, so the healthcheck MUST be exec-form calling the
# binary directly. `server healthcheck` GETs http://127.0.0.1:$PORT/healthz
# and exits 0/1. Compose honors this image healthcheck for service_healthy.
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD ["/server", "healthcheck"]
ENTRYPOINT ["/server"]
