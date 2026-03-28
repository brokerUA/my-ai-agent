FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy dependency manifests
COPY src/go.mod ./
# (If go.sum is present, copy it as well)
COPY src/go.sum* ./

# Copy local modules (needed for go mod download due to replace directives)
COPY src/adk-local/ ./adk-local/

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Build the agent binary
RUN CGO_ENABLED=0 GOOS=linux go build -o agent main.go

# Use a minimal runtime image
FROM alpine:3.18

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/agent .

# Expose any necessary ports (assuming 8080 as standard)
EXPOSE 8080

# Command to run the agent
CMD ["./agent"]
