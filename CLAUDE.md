# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go microservice that provides a unified REST API for deploying and managing both containerized applications (via Kubernetes Deployments) and virtual machines (via KubeVirt). The service abstracts the complexity of Kubernetes and KubeVirt operations behind a simple API with global deployment management across namespaces.

## Build and Development Commands

### Building
```bash
make build                    # Build for current platform
make build-linux             # Build for Linux AMD64
make build-darwin            # Build for macOS AMD64
make build-all               # Build for all platforms
```

### Testing
```bash
make test                    # Run all tests with race detector
make test-unit               # Run only unit tests (internal/...)
make test-integration        # Run only integration tests (test/...)
make test-coverage           # Generate coverage report (creates coverage/coverage.html)
make test-short              # Run tests with -short flag
go test -run TestName ./internal/package  # Run specific test
```

### Code Quality
```bash
make lint                    # Run pre-commit hooks (golangci-lint, markdownlint, shellcheck)
make fmt                     # Format all Go code
make vet                     # Run go vet
make sec                     # Run gosec security scanner
make check                   # Run fmt + vet + lint + sec + test (full validation)
```

### Development
```bash
make dev                     # Run server with hot reload (go run cmd/server/main.go)
make deps                    # Download and verify dependencies
make clean                   # Clean build artifacts and cache
```

### Container Operations
```bash
make image-build             # Build container image with podman
make image-run               # Run container (maps ports for both services)
make image-stop              # Stop running container
```

## Architecture

### Two-Service Design
The application runs two HTTP servers concurrently:
- **Deployment Service**: Manages container and VM deployments
- **Namespace Service**: Queries namespaces by label selectors

Both servers share a single Kubernetes client instance and use graceful shutdown. The deployment service runs on the configured SERVER_PORT (default defined in config), and the namespace service runs on port 8081.

### Core Components

1. **API Layer** (`internal/{deployment,namespace}/api/`)
   - HTTP handlers using Gin framework
   - Request validation and error response formatting
   - Router setup with health check endpoints

2. **Service Layer** (`internal/{deployment,namespace}/services/`)
   - `DeploymentService`: Orchestrates between ContainerService and VMService
   - `ContainerService`: Manages Kubernetes Deployments and Services
   - `VMService`: Manages KubeVirt VirtualMachine resources
   - `NamespaceService`: Queries namespaces by labels

3. **Models** (`internal/deployment/models/`)
   - Type-safe request/response models
   - Custom error types: `ErrDeploymentNotFound`, `ErrMultipleDeploymentsFound`, `ErrDeploymentAlreadyExists`
   - Label constants and helper functions

4. **K8s Client** (`internal/k8s/`)
   - Wrapper around client-go with interface for testability
   - Supports both in-cluster and kubeconfig-based authentication
   - Health check and namespace query operations

5. **Configuration** (`internal/config/`)
   - Environment variable-based configuration
   - Validation on startup
   - Server, Kubernetes, and logging configuration

### Global Deployment Management

**Critical architectural pattern**: Deployment IDs are globally unique across all namespaces.

- Each deployment gets a UUID assigned on creation
- GET/PUT/DELETE by ID automatically search ALL namespaces to find the deployment
- Only POST (create) requires namespace to be specified
- `GetDeploymentByID()` searches both container and VM services across all namespaces
- Returns `ErrMultipleDeploymentsFound` if same ID exists multiple times (data integrity violation)

Implementation in `internal/deployment/services/service.go` (GetDeploymentByID function):
```go
func (d *DeploymentService) GetDeploymentByID(ctx context.Context, id string) (*models.DeploymentResponse, error) {
    var foundDeployments []*models.DeploymentResponse

    // Try to find as container
    if deployment, err := d.containerService.GetContainer(ctx, id); err == nil {
        foundDeployments = append(foundDeployments, deployment)
    }

    // Try to find as VM
    if deployment, err := d.vmService.GetVM(ctx, id); err == nil {
        foundDeployments = append(foundDeployments, deployment)
    }

    // Check for conflicts and return appropriately
}
```

### Resource Labeling System

All managed resources use standardized labels (see `internal/deployment/models/deployment.go`):

**Label Constants**:
- `LabelManagedBy` = "managed-by" (value: "k8s-service-provider")
- `LabelAppID` = "app-id" (value: deployment UUID)
- `LabelApp` = "app" (value: deployment name)

**Helper Functions**:
- `BuildDeploymentLabels(id, name)`: Creates standard label map
- `BuildDeploymentSelector(id)`: Creates label selector for specific deployment
- `BuildManagedResourceSelector()`: Creates label selector for all managed resources

These labels enable:
- Filtering resources managed by this service vs other controllers
- Precise selection of resources for a specific deployment
- Global deployment ID lookups across namespaces

### Error Handling

Custom error types provide type-safe error handling with proper HTTP status mapping:

- `ErrDeploymentNotFound`: 404 - Deployment not found (in namespace or globally)
- `ErrMultipleDeploymentsFound`: 409 - Data integrity violation (multiple deployments with same ID)
- `ErrDeploymentAlreadyExists`: 409 - Cannot create deployment with existing ID

Helper functions in `internal/deployment/models/errors.go`:
- `IsNotFoundError(err)`: Check if error is not found
- `IsMultipleFoundError(err)`: Check if multiple found
- `IsConflictError(err)`: Check if conflict (already exists or multiple found)
- `IsAlreadyExistsError(err)`: Check if already exists

### Deployment Request Routing

`DeploymentService` routes operations by `kind` field ("container" or "vm"):
- Create/Update/Delete: Calls appropriate service method based on `kind`
- GetByID: Searches both services (global lookup)
- List: Aggregates results from both services if no kind filter specified

## Testing Strategy

### Test Organization
- **Unit tests**: Located alongside source files (`*_test.go`)
- **Integration tests**: In `test/` directory
- Use table-driven tests where applicable
- Mock Kubernetes clientset using interfaces

### Running Specific Tests
```bash
# Run tests for a specific package
go test ./internal/deployment/services

# Run a specific test function
go test -run TestDeploymentService_CreateDeployment ./internal/deployment/services

# Run with verbose output
go test -v ./internal/deployment/services

# Run with race detection and coverage
go test -race -coverprofile=coverage.out ./internal/deployment/services
```

## Configuration

Configuration is managed through environment variables. Default values are defined in `internal/config/config.go` (LoadConfig function).

### Server Configuration
- `SERVER_PORT`: HTTP server port
- `SERVER_HOST`: HTTP server host
- `SERVER_READ_TIMEOUT`: Read timeout in seconds
- `SERVER_WRITE_TIMEOUT`: Write timeout in seconds

### Kubernetes Configuration
- `KUBECONFIG`: Path to kubeconfig file
- `IN_CLUSTER`: Use in-cluster configuration

### Logging Configuration
- `LOG_LEVEL`: Log level (debug, info, warn, error)
- `LOG_FORMAT`: Log format (json, console)
- `LOG_OUTPUT_PATH`: Log output path

See `internal/config/config.go` for current default values.

## API Endpoints Reference

See `api/openapi.yaml` and `api/namespace-openapi.yaml` for complete API documentation.

### Deployment Service
- `POST /api/v1/deployments` - Create deployment (namespace required in body)
- `GET /api/v1/deployments` - List deployments with filtering
- `GET /api/v1/deployments/{id}` - Get deployment by ID (searches globally)
- `PUT /api/v1/deployments/{id}` - Update deployment by ID
- `DELETE /api/v1/deployments/{id}` - Delete deployment by ID
- `GET /api/v1/health` - Health check

### Namespace Service
- `POST /api/v1/namespaces` - Get namespaces by label selectors
- `GET /api/v1/health` - Health check

The deployment service runs on SERVER_PORT (configurable via environment, see Configuration section). The namespace service runs on port 8081.

## Key Implementation Details

### Deployment Kinds
Two deployment kinds are supported (defined in `internal/deployment/models/deployment.go`):
- `"container"`: Creates Kubernetes Deployment + Service
- `"vm"`: Creates KubeVirt VirtualMachine

### Container Deployments
- Creates Kubernetes Deployment with configurable replicas
- Creates Kubernetes Service for port exposure (if ports specified)
- Supports environment variables via `spec.container.environment`
- Resources specified as `cpu` and `memory` strings (e.g., "100m", "128Mi")

### VM Deployments
- Creates KubeVirt VirtualMachine resources
- Supported OS: fedora, ubuntu, centos, rhel
- RAM: 1-32 GB
- CPU: 1-32 cores
- Uses containerDisk for OS images

### OpenAPI Specifications
API documentation is in `api/openapi.yaml` and `api/namespace-openapi.yaml`.

## Code Quality Tools

The project uses pre-commit hooks (`.pre-commit-config.yaml`):
- **golangci-lint**: Go linting (timeout: 5m)
- **markdownlint**: Markdown linting (ignores MD013, MD002)
- **shellcheck**: Shell script linting
- **check-json**: JSON validation
- **check-yaml**: YAML validation
- **trailing-whitespace**: Whitespace cleanup
- **end-of-file-fixer**: EOF normalization

Install pre-commit hooks: `make install-tools`

## Git Commit Requirements

All commits to this repository must include a Developer Certificate of Origin (DCO) sign-off.

### How to Sign Off Commits

Always use the `-s` or `--signoff` flag when committing:

```bash
git commit -s -m "Your commit message"
```

This automatically adds a `Signed-off-by` line with your name and email:
```
Signed-off-by: Your Name <your.email@example.com>
```

### What This Means

By signing off, you certify that:
1. You wrote the contribution or have the right to submit it
2. You understand it will be distributed under the project's license
3. You comply with the Developer Certificate of Origin (DCO)

### Important Notes

- **Never commit without sign-off**: All commits require the `-s` flag
- **Amending commits**: When amending, use `git commit --amend -s` to ensure sign-off is preserved
- **CI/CD checks**: PRs may be rejected if commits lack proper sign-off

## Important Patterns and Conventions

1. **Always use label constants**: Never hardcode "managed-by" or "app-id" strings. Use `models.LabelManagedBy`, `models.LabelAppID`, etc.

2. **Use helper functions for label operations**: Use `BuildDeploymentLabels()`, `BuildDeploymentSelector()` instead of manually building label maps/selectors.

3. **Check error types properly**: Use `models.IsNotFoundError()`, `models.IsConflictError()` instead of string matching.

4. **Global ID uniqueness**: When implementing create operations, always call `GetDeploymentByID()` first to verify ID doesn't exist globally.

5. **Structured logging**: Use zap logger with contextual fields. Example:
   ```go
   logger.Info("Creating deployment",
       zap.String("kind", string(req.Kind)),
       zap.String("deployment_id", id))
   ```

6. **Interface-based design**: All services depend on interfaces (`ClientInterface`, `DeploymentServiceInterface`) for testability.

7. **Context propagation**: Always pass `context.Context` through the call stack for cancellation support.
