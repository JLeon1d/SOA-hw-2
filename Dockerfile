# Build stage
FROM golang:alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate code from OpenAPI spec
RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest && \
    mkdir -p internal/generated/api && \
    oapi-codegen -package api -generate types,server,spec -o internal/generated/api/api.gen.go api/openapi/marketplace.yaml

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o marketplace-server cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS and debugging tools
RUN apk --no-cache add ca-certificates postgresql-client netcat-openbsd bind-tools

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/marketplace-server .

# Copy test script
COPY test_connection.sh .
RUN chmod +x test_connection.sh

# Expose port
EXPOSE 8080

# Run the application
CMD ["./marketplace-server"]