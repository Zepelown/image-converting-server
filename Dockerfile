# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o image-server main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/image-server .
# Copy default config (if needed)
COPY --from=builder /app/config ./config

# Create data directory for state
RUN mkdir -p data

# Expose port (default 4000, can be overridden by env)
EXPOSE 4000

# Command to run the application
CMD ["./image-server"]
