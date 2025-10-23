# DCM Service Provider

A microservice for managing container and virtual machine deployments on Kubernetes with KubeVirt support.

## Overview

This microservice provides a unified API for deploying and managing both containerized applications and virtual machines. It abstracts the complexity of Kubernetes and KubeVirt deployments behind a simple REST API.

## Features

- **Unified API**: Single endpoint with CRUD operations for both containers and VMs
- **Container Deployments**: Deploy containerized applications using Kubernetes Deployments and Services
- **Virtual Machine Deployments**: Deploy VMs using KubeVirt with configurable resources
- **Automatic Routing**: Request routing based on deployment kind (container vs VM)
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

## API Endpoints

- `POST /api/v1/deployments` - Create a new deployment
- `GET /api/v1/deployments` - List deployments with filtering
- `GET /api/v1/deployments/{id}` - Get specific deployment
- `PUT /api/v1/deployments/{id}` - Update deployment
- `DELETE /api/v1/deployments/{id}` - Delete deployment
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
      }
    }
  }
}
```

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

## Prerequisites

- Go 1.23 or later
- Kubernetes cluster with KubeVirt installed (for VM deployments)
- Valid kubeconfig for cluster access

## Installation

### From Source

1. Clone the repository:
```bash
git clone <repository-url>
cd dcm-service-provider
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
./bin/dcm-service-provider
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
dcm-service-provider/
├── api/                    # OpenAPI specifications
├── cmd/server/            # Application entrypoint
├── internal/              # Internal packages
│   ├── api/              # HTTP handlers and routing
│   ├── config/           # Configuration management
│   ├── deploy/           # Deployment services
│   └── models/           # Data models
├── test/                  # Integration tests
├── Containerfile         # Container build instructions
├── Makefile              # Build automation
├── go.mod                # Go module definition
└── README.md             # This file
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `make check`
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For questions or issues, please:
1. Check the existing issues in the repository
2. Create a new issue with detailed information
3. Contact the DCM team at dcm-team@example.com