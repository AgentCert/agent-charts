# Build stage
# Base images: upstream names by default (plain `docker build` works anywhere).
# scripts/build-and-push.sh, setup.sh and prepare-images.sh pass the copy from
# IMAGE_REGISTRY / the frozen agentcert/ Docker Hub copies (scripts/lib/registry.sh).
ARG GOLANG_IMAGE=golang:1.21-alpine3.19
ARG ALPINE_IMAGE=alpine:3.19
FROM ${GOLANG_IMAGE} AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY install-agent/go.mod ./
RUN go mod download

# Copy source code
COPY install-agent/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o install-agent .

# Final stage
FROM ${ALPINE_IMAGE}

# Install helm and kubectl
RUN apk add --no-cache \
    curl \
    bash \
    ca-certificates \
    && curl -fsSL https://get.helm.sh/helm-v3.14.0-linux-amd64.tar.gz | tar -xzv \
    && mv linux-amd64/helm /usr/local/bin/helm \
    && rm -rf linux-amd64 \
    && curl -fsSL "https://dl.k8s.io/release/v1.29.0/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/install-agent /usr/local/bin/install-agent

# Copy all chart repositories into the container
COPY charts/ /charts/

# Set permissions
RUN chmod +x /usr/local/bin/install-agent && \
    chown -R appuser:appuser /app /charts

USER appuser

# Default entrypoint
ENTRYPOINT ["install-agent"]

# Default command shows help
CMD ["--help"]
