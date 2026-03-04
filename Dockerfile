# Build stage
FROM golang:alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

# Generate OpenAPI spec
RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest && \
    mkdir -p internal/generated/api && \
    oapi-codegen -package api -generate types,server,spec -o internal/generated/api/api.gen.go api/openapi/marketplace.yaml

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o marketplace-server cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS and debugging tools
RUN apk --no-cache add ca-certificates postgresql-client netcat-openbsd bind-tools

WORKDIR /root/

COPY --from=builder /app/marketplace-server .
COPY --from=builder /app/migrations ./migrations

# Copy test script
COPY test_connection.sh .
RUN chmod +x test_connection.sh

EXPOSE 8080

CMD ["./marketplace-server"]
