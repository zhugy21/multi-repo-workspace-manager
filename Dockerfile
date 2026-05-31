# ── Stage 1: Build ───────────────────────────────────────────
FROM golang:1.24-alpine AS builder

ARG http_proxy
ARG https_proxy
ENV http_proxy=$http_proxy \
    https_proxy=$https_proxy \
    GOPROXY=direct \
    GONOSUMDB="*"

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/ws ./cmd/ws/

# ── Stage 2: Runtime ──────────────────────────────────────────
FROM alpine:3.21

ARG http_proxy
ARG https_proxy
ENV http_proxy=$http_proxy \
    https_proxy=$https_proxy

RUN apk add --no-cache git

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/local/bin/ws /usr/local/bin/ws

ENTRYPOINT ["ws"]
