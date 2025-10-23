# Build stage
FROM registry.access.redhat.com/ubi9/go-toolset:9.6 AS builder

# Switch to root to have write permissions
USER root

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o k8s-service-provider cmd/server/main.go

# Final stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

# Install ca-certificates for HTTPS calls (curl-minimal is already present)
RUN microdnf install -y ca-certificates && microdnf clean all

# Create a non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -s /bin/bash -m appuser

# Set working directory to appuser's home
WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /app/k8s-service-provider .

# Copy OpenAPI spec (for documentation)
COPY --from=builder /app/api/openapi.yaml .

# Create .kube directory for kubeconfig mounting
RUN mkdir -p /home/appuser/.kube

# Change ownership to non-root user
RUN chown -R appuser:appgroup /home/appuser

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Labels for metadata
LABEL maintainer="K8s Team <k8s-team@example.com>" \
      org.opencontainers.image.title="K8S Service Provider for DCM" \
      org.opencontainers.image.description="A microservice for managing container and virtual machine deployments" \
      org.opencontainers.image.vendor="DCM Project" \
      org.opencontainers.image.source="https://github.com/dcm/k8s-service-provider" \
      org.opencontainers.image.licenses="MIT"

# Run the application
CMD ["./k8s-service-provider"]