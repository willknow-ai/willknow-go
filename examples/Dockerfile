FROM golang:1.21 AS builder

WORKDIR /src

# Copy go mod files from parent directory
COPY ../go.mod ../go.sum ./
RUN go mod download

# Copy all source code (including the library and examples)
COPY .. .

# Build the application
WORKDIR /src/examples
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/myapp main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Copy the binary
COPY --from=builder /app/myapp .

# Copy source code (for AI Assistant to analyze)
COPY --from=builder /src /app/source

# Create log directory
RUN mkdir -p /var/log

# Expose ports
# 8080: Main application
# 8888: AI Assistant
EXPOSE 8080 8888

# Run the application
CMD ["./myapp"]
