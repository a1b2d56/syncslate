# Stage 1: Build binary
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/syncslate ./cmd/syncslate

# Stage 2: Minimal runtime container
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/syncslate /app/syncslate
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/.env.example /app/.env.example

EXPOSE 8080

ENTRYPOINT ["/app/syncslate"]
