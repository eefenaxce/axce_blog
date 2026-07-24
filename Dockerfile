# syntax=docker/dockerfile:1

# Stage 1: Build frontend assets
FROM node:24-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@latest --activate
RUN pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm build

# Stage 2: Build Go backend
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/build/client ./web/build/client
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 3: Runtime image
FROM alpine:latest
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=go-builder /app/server .
COPY --from=go-builder /app/web/build/client ./web/build/client
COPY --from=go-builder /app/sqlc ./sqlc

EXPOSE 8080
VOLUME ["/app/themes"]

ENTRYPOINT ["./server"]