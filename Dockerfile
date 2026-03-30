# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/wallet ./cmd/main.go

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
# distroless has no shell — dramatically reduces attack surface
FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/wallet /app/wallet

EXPOSE 8080

ENTRYPOINT ["/app/wallet"]