# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cortex .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/cortex .

# Create directory for database
RUN mkdir -p /data

# Expose port
EXPOSE 9123

# Set environment variables
ENV CORTEX_PORT=9123
ENV CORTEX_DB_PATH=/data/cortex.db

# Run the binary
CMD ["./cortex"]
