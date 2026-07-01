# syntax=docker/dockerfile:1

# Build stage
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
ENTRYPOINT ["/server"]
