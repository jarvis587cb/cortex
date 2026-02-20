# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . ./

# Build server binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cortex-server ./cmd/cortex-server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/cortex-server .

# Create directory for database
RUN mkdir -p /data

# Expose port
EXPOSE 9123

# Run the server
CMD ["./cortex-server"]
