# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -a -installsuffix cgo \
    -o paymentservice ./cmd/paymentservice

# Final stage - distroless image
FROM gcr.io/distroless/static-debian11:nonroot

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/paymentservice .

# Expose port
EXPOSE 50051

# Run the application
CMD ["/app/paymentservice"]
