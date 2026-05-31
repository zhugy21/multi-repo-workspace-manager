# ── Stage 1: Build ───────────────────────────────────────────
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/ws ./cmd/ws/

# ── Stage 2: Runtime ──────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache git ca-certificates

COPY --from=builder /usr/local/bin/ws /usr/local/bin/ws

ENTRYPOINT ["ws"]
