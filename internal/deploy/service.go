package deploy

import (
	"context"
	"fmt"

	"github.com/dcm/service-provider/internal/models"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

// DeploymentService orchestrates container and VM deployments
type DeploymentService struct {
	containerService *ContainerService
	vmService        *VMService
	logger           *zap.Logger
}

// NewDeploymentService creates a new deployment service
func NewDeploymentService(k8sClient kubernetes.Interface, logger *zap.Logger) *DeploymentService {
	return &DeploymentService{
		containerService: NewContainerService(k8sClient, logger),
		vmService:        NewVMService(k8sClient, logger),
		logger:           logger,
	}
}

// CreateDeployment creates a new deployment based on the kind
func (d *DeploymentService) CreateDeployment(ctx context.Context, req *models.DeploymentRequest, id string) error {
	logger := d.logger.Named("deployment_service").With(
		zap.String("kind", string(req.Kind)),
		zap.String("name", req.Metadata.Name),
		zap.String("deployment_id", id),
	)

	logger.Info("Creating deployment")

	switch req.Kind {
	case models.DeploymentKindContainer:
		return d.containerService.CreateContainer(ctx, req, id)
	case models.DeploymentKindVM:
		return d.vmService.CreateVM(ctx, req, id)
	default:
		return fmt.Errorf("unsupported deployment kind: %s", req.Kind)
	}
}

// GetDeployment retrieves a deployment by ID and kind
func (d *DeploymentService) GetDeployment(ctx context.Context, id, namespace string, kind models.DeploymentKind) (*models.DeploymentResponse, error) {
	logger := d.logger.Named("deployment_service").With(
		zap.String("kind", string(kind)),
		zap.String("deployment_id", id),
	)

	logger.Info("Getting deployment")

	switch kind {
	case models.DeploymentKindContainer:
		return d.containerService.GetContainer(ctx, id, namespace)
	case models.DeploymentKindVM:
		return d.vmService.GetVM(ctx, id, namespace)
	default:
		return nil, fmt.Errorf("unsupported deployment kind: %s", kind)
	}
}

// UpdateDeployment updates an existing deployment
func (d *DeploymentService) UpdateDeployment(ctx context.Context, req *models.DeploymentRequest, id string) error {
	logger := d.logger.Named("deployment_service").With(
		zap.String("kind", string(req.Kind)),
		zap.String("name", req.Metadata.Name),
		zap.String("deployment_id", id),
	)

	logger.Info("Updating deployment")

	switch req.Kind {
	case models.DeploymentKindContainer:
		return d.containerService.UpdateContainer(ctx, req, id)
	case models.DeploymentKindVM:
		return d.vmService.UpdateVM(ctx, req, id)
	default:
		return fmt.Errorf("unsupported deployment kind: %s", req.Kind)
	}
}

// DeleteDeployment deletes a deployment by ID and kind
func (d *DeploymentService) DeleteDeployment(ctx context.Context, id, namespace string, kind models.DeploymentKind) error {
	logger := d.logger.Named("deployment_service").With(
		zap.String("kind", string(kind)),
		zap.String("deployment_id", id),
	)

	logger.Info("Deleting deployment")

	switch kind {
	case models.DeploymentKindContainer:
		return d.containerService.DeleteContainer(ctx, id, namespace)
	case models.DeploymentKindVM:
		return d.vmService.DeleteVM(ctx, id, namespace)
	default:
		return fmt.Errorf("unsupported deployment kind: %s", kind)
	}
}

// ListDeployments lists deployments with filtering and pagination
func (d *DeploymentService) ListDeployments(ctx context.Context, req *models.ListDeploymentsRequest) (*models.ListDeploymentsResponse, error) {
	logger := d.logger.Named("deployment_service").With(
		zap.String("namespace", req.Namespace),
		zap.String("kind", string(req.Kind)),
		zap.Int("limit", req.Limit),
		zap.Int("offset", req.Offset),
	)

	logger.Info("Listing deployments")

	var allDeployments []models.DeploymentResponse

	// List containers if kind is empty or container
	if req.Kind == "" || req.Kind == models.DeploymentKindContainer {
		containers, err := d.containerService.ListContainers(ctx, req.Namespace, req.Limit, 0)
		if err != nil {
			logger.Error("Failed to list containers", zap.Error(err))
			return nil, fmt.Errorf("failed to list containers: %w", err)
		}
		allDeployments = append(allDeployments, containers...)
	}

	// List VMs if kind is empty or vm
	if req.Kind == "" || req.Kind == models.DeploymentKindVM {
		vms, err := d.vmService.ListVMs(ctx, req.Namespace, req.Limit, 0)
		if err != nil {
			logger.Error("Failed to list VMs", zap.Error(err))
			return nil, fmt.Errorf("failed to list VMs: %w", err)
		}
		allDeployments = append(allDeployments, vms...)
	}

	// Apply pagination
	total := len(allDeployments)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		allDeployments = []models.DeploymentResponse{}
	} else {
		if end > total {
			end = total
		}
		allDeployments = allDeployments[start:end]
	}

	response := &models.ListDeploymentsResponse{
		Deployments: allDeployments,
		Pagination: models.Pagination{
			Limit:   req.Limit,
			Offset:  req.Offset,
			Total:   total,
			HasMore: req.Offset+req.Limit < total,
		},
	}

	logger.Info("Successfully listed deployments", zap.Int("count", len(allDeployments)))
	return response, nil
}

// GetDeploymentByID retrieves a deployment by ID, searching both containers and VMs
func (d *DeploymentService) GetDeploymentByID(ctx context.Context, id, namespace string) (*models.DeploymentResponse, error) {
	logger := d.logger.Named("deployment_service").With(zap.String("deployment_id", id))

	// Try to find as container first
	if deployment, err := d.containerService.GetContainer(ctx, id, namespace); err == nil {
		return deployment, nil
	}

	// Try to find as VM
	if deployment, err := d.vmService.GetVM(ctx, id, namespace); err == nil {
		return deployment, nil
	}

	logger.Warn("Deployment not found", zap.String("deployment_id", id))
	return nil, fmt.Errorf("deployment not found")
}