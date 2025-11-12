# Multi-stage build

# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
RUN go build -o ras-grpc-gw cmd/main.go

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /root/

COPY --from=builder /app/ras-grpc-gw .

# Expose gRPC port
EXPOSE 9999

# Health check (HTTP health endpoint на порту 8080)
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["./ras-grpc-gw", "--host", "0.0.0.0:9999", "--ras", "host.docker.internal:1545"]
