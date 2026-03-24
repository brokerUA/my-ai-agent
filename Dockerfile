# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o agent-app main.go

# Final stage
FROM alpine:latest

# Create a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/agent-app .

# Use non-root user
USER appuser

# Expose the port the agent listens on
EXPOSE 8080

# Run the application
CMD ["./agent-app"]