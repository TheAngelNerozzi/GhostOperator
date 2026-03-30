# GHOSTOPERATOR (GO) - Dockerized Orchestrator
# Use multi-stage builds for minimal image size

# STAGE 1: Build Phase
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy project source
COPY . .

# Build the orchestrator binary (targeting linux)
# Note: Full automation (mouse/keyboard) requires Windows host, 
# but the dashboard and mission logic run natively on Linux containers.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o ghost-orchestrator ./cmd/ghost

# STAGE 2: Runtime Phase
FROM alpine:latest

# Install minimal certificates for external VLM communication (Ollama)
RUN apk add --no-cache ca-certificates

WORKDIR /root/

# Copy the binary from the build stage
COPY --from=builder /app/ghost-orchestrator .

# Copy the web dashboard and configuration
COPY --from=builder /app/landing_page ./landing_page
COPY --from=builder /app/config.json .

# Default configuration ports
# DASHBOARD: 7474
EXPOSE 7474

# Expose OLLAMA_HOST for redirection to a local or sidecar instance
ENV OLLAMA_HOST=http://host.docker.internal:11434

# Start the orchestrator in terminal mode (start)
CMD ["./ghost-orchestrator", "start"]
