# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy module manifest first — leverages Docker layer cache for dependencies
COPY go.mod ./
RUN go mod download || true

# Copy full source and tidy (generates go.sum if absent), then build
COPY . .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-w -s" \
      -o /server \
      ./cmd/server

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM alpine:3.20

# ca-certificates: needed for outbound TLS; tzdata: for time-zone support
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /server /app/server

EXPOSE 8080

CMD ["/app/server"]
