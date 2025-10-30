# K8s Service Provider

A microservice for managing container and virtual machine deployments on Kubernetes with KubeVirt support.

## Overview

This microservice provides a unified API for deploying and managing both containerized applications and virtual machines. It abstracts the complexity of Kubernetes and KubeVirt deployments behind a simple REST API.

## Features

- **Unified API**: Single endpoint with CRUD operations for both containers and VMs
- **Container Deployments**: Deploy containerized applications using Kubernetes Deployments and Services
- **Virtual Machine Deployments**: Deploy VMs using KubeVirt with configurable resources
- **Global Deployment Management**: Unique deployment IDs across all namespaces with automatic lookup
- **Resource Filtering**: Managed resource tracking with standardized labels for easy identification
- **Environment Variables**: Full support for environment variable configuration in container deployments
- **Automatic Routing**: Request routing based on deployment kind (container vs VM)
- **Custom Error Types**: Type-safe error handling with specific error codes and messages
- **Comprehensive Testing**: Unit tests, integration tests, and API tests
- **OpenAPI Specification**: Complete API documentation with OpenAPI 3.0
- **Cloud-Native**: Built for Kubernetes with proper health checks and logging

## Architecture

The service consists of several key components:

- **API Layer**: HTTP handlers and routing using Gin framework
- **Deployment Service**: Orchestrates container and VM deployments
- **Container Service**: Manages Kubernetes deployments and services
- **VM Service**: Manages KubeVirt virtual machines
- **Configuration**: Environment-based configuration management

## Technical Improvements

### Global Deployment Management
The service implements global deployment ID uniqueness across all Kubernetes namespaces:
- **Unique IDs**: Each deployment gets a globally unique UUID, preventing conflicts across namespaces
- **Automatic Lookup**: GET, PUT, and DELETE operations automatically search all namespaces to find deployments by ID
- **Cross-Namespace Operations**: No need to specify namespace for individual deployment operations

### Resource Filtering and Management
All managed resources are tagged with standardized labels for easy identification and filtering:
- **Managed-By Labels**: All resources include `managed-by=k8s-service-provider` for identification
- **App ID Labels**: Each deployment includes `app-id=<deployment-uuid>` for precise resource selection
- **Centralized Constants**: Label names and values are defined as constants in code for maintainability

### Enhanced Error Handling
Type-safe error handling with specific error types and HTTP status codes:
- **Custom Error Types**: `DeploymentNotFoundError`, `ConflictError`, `MultipleFoundError`
- **Consistent API Responses**: Standardized error response format with error codes
- **Proper HTTP Status Codes**: Accurate status codes for different error conditions

### Environment Variable Support
Full support for environment variable configuration in container deployments:
- **Multiple Variables**: Support for setting multiple environment variables per container
- **Flexible Configuration**: Name-value pairs for runtime configuration
- **Template Integration**: Environment variables are properly integrated into Kubernetes Deployment templates

## API Endpoints

### Deployment Service (Port 8080)
- `POST /api/v1/deployments` - Create a new deployment (namespace required)
- `GET /api/v1/deployments` - List deployments with filtering (searches all namespaces)
- `GET /api/v1/deployments/{id}` - Get specific deployment by ID (searches globally)
- `PUT /api/v1/deployments/{id}` - Update deployment by ID (auto-detects namespace)
- `DELETE /api/v1/deployments/{id}` - Delete deployment by ID (auto-detects namespace and kind)
- `GET /api/v1/health` - Health check

**Note**: The service implements global deployment ID uniqueness. GET, PUT, and DELETE operations automatically search across all namespaces to find deployments by ID, eliminating the need to specify namespace parameters for these operations.

### Namespace Service (Port 8081)
- `POST /api/v1/namespaces` - Get namespaces by label selectors
- `GET /api/v1/health` - Health check

## Deployment Types

### Container Deployments

```json
{
  "kind": "container",
  "metadata": {
    "name": "nginx-app",
    "namespace": "default",
    "labels": {
      "app": "nginx"
    }
  },
  "spec": {
    "container": {
      "image": "nginx:latest",
      "replicas": 3,
      "ports": [
        {
          "containerPort": 80,
          "servicePort": 8080
        }
      ],
      "resources": {
        "cpu": "100m",
        "memory": "128Mi"
      },
      "environment": [
        {
          "name": "ENV_NAME",
          "value": "production"
        },
        {
          "name": "DEBUG_MODE",
          "value": "false"
        }
      ]
    }
  }
}
```

**Container Configuration Options**:
- `image`: Container image to deploy (required)
- `replicas`: Number of pod replicas (optional, default: 1)
- `ports`: Port configurations for container and service exposure
- `resources`: CPU and memory resource requests
- `environment`: Environment variables to set in the container

### Virtual Machine Deployments

```json
{
  "kind": "vm",
  "metadata": {
    "name": "fedora-vm",
    "namespace": "default",
    "labels": {
      "os": "fedora",
      "environment": "development"
    }
  },
  "spec": {
    "vm": {
      "ram": 4,
      "cpu": 2,
      "os": "fedora"
    }
  }
}
```

**Supported Operating Systems**: `fedora`, `ubuntu`, `centos`, `rhel`

**VM Specifications**:
- `ram`: Memory allocation in GB (1-32)
- `cpu`: Number of CPU cores (1-32)
- `os`: Operating system (automatically maps to appropriate container disk image)

### Namespace Queries

```json
{
  "labels": {
    "environment": "production",
    "team": "backend"
  }
}
```

**Response Example**:
```json
{
  "namespaces": [
    {
      "name": "production-backend",
      "labels": {
        "environment": "production",
        "team": "backend",
        "created-by": "ops"
      }
    }
  ],
  "count": 1
}
```

## Prerequisites

- Go 1.23 or later
- Kubernetes cluster with KubeVirt installed (for VM deployments)
- Valid kubeconfig for cluster access

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/dcm-project/k8s-service-provider.git
cd k8s-service-provider
```

2. Install dependencies:
```bash
make deps
```

3. Build the application:
```bash
make build
```

4. Run the application:
```bash
./bin/k8s-service-provider
```

### Using Container

1. Build the container image:
```bash
make image-build
```

2. Run the container:
```bash
make image-run
```

3. Stop the container:
```bash
make image-stop
```


## Configuration

The application can be configured using environment variables:

### Server Configuration
- `SERVER_PORT` - HTTP server port (default: 8080)
- `SERVER_HOST` - HTTP server host (default: 0.0.0.0)
- `SERVER_READ_TIMEOUT` - Read timeout in seconds (default: 30)
- `SERVER_WRITE_TIMEOUT` - Write timeout in seconds (default: 30)

### Kubernetes Configuration
- `KUBECONFIG` - Path to kubeconfig file (default: ~/.kube/config)
- `IN_CLUSTER` - Use in-cluster configuration (default: false)

### Logging Configuration
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `LOG_FORMAT` - Log format: json, console (default: json)
- `LOG_OUTPUT_PATH` - Log output path (default: stdout)

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test types
make test-unit
make test-integration
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run security scanner
make sec

# Run all checks
make check
```

### Development Server

```bash
# Run with hot reload
make dev
```

### Additional Development Commands

```bash
# Install dependencies
make deps

# Clean build artifacts and cache
make clean

# Install development tools (linter, security scanner, etc.)
make install-tools

# Show build information
make info

# Run benchmarks
make benchmark
```

## API Documentation

The OpenAPI specification is available at `/api/openapi.yaml`. You can serve the documentation using:

```bash
make docs-serve
```

## Building and Deployment

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for specific platforms
make build-linux
make build-windows
make build-darwin
```

### Container Commands

```bash
# Build container image
make image-build

# Run container
make image-run

# Stop container
make image-stop
```


## Project Structure

```
k8s-service-provider/
├── api/                         # OpenAPI specifications
├── cmd/server/                  # Application entrypoint
├── internal/                    # Internal packages
│   ├── config/                 # Configuration management
│   ├── k8s/                    # Kubernetes client wrapper
│   ├── deployment/             # Deployment-related functionality
│   │   ├── api/               # HTTP handlers and routing
│   │   ├── models/            # Data models and error types
│   │   └── services/          # Business logic services
│   └── namespace/              # Namespace-related functionality
│       ├── api/               # HTTP handlers for namespace operations
│       ├── models/            # Namespace models
│       └── services/          # Namespace services
├── test/                        # Integration tests
├── Containerfile               # Container build instructions
├── Makefile                    # Build automation
├── go.mod                      # Go module definition
├── LICENSE                     # Apache 2.0 license
└── README.md                   # This file
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `make check`
6. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

## Support

For questions or issues, please:
1. Check the existing issues in the repository
2. Create a new issue with detailed information
3. Refer to the project documentation and examples
