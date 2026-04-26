# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -o collector ./shell/server/main.go

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/collector .

# Expose gRPC and Prometheus metrics ports
EXPOSE 9001
EXPOSE 9100

# Set environment variables (can be overridden in docker-compose)
ENV KAFKA_BROKERS=redpanda:9092
ENV GRPC_PORT=9001
ENV METRICS_PORT=9100

CMD ["./collector"]
