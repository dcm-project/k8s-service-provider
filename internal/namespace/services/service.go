package services

import (
	"context"

	"github.com/dcm/k8s-service-provider/internal/k8s"
	"github.com/dcm/k8s-service-provider/internal/namespace/models"
	"go.uber.org/zap"
)

// NamespaceService handles namespace operations
type NamespaceService struct {
	k8sClient k8s.ClientInterface
	logger    *zap.Logger
}

// NewNamespaceService creates a new namespace service instance
func NewNamespaceService(k8sClient k8s.ClientInterface, logger *zap.Logger) *NamespaceService {
	return &NamespaceService{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetNamespacesByLabels retrieves namespaces that match the provided label selectors
func (s *NamespaceService) GetNamespacesByLabels(ctx context.Context, labelSelectors map[string]string) (*models.NamespaceResponse, error) {
	s.logger.Info("Processing label selectors", zap.Any("labels", labelSelectors))

	// Get namespaces from Kubernetes using shared client
	namespaceInfos, err := s.k8sClient.GetNamespacesByLabels(ctx, labelSelectors)
	if err != nil {
		s.logger.Error("Failed to get namespaces from Kubernetes", zap.Error(err))
		return nil, err
	}

	// Convert to namespace response format
	namespaces := make([]models.Namespace, 0, len(namespaceInfos))
	for _, nsInfo := range namespaceInfos {
		namespace := models.Namespace{
			Name:   nsInfo.Name,
			Labels: nsInfo.Labels,
		}
		namespaces = append(namespaces, namespace)
	}

	response := &models.NamespaceResponse{
		Namespaces: namespaces,
		Count:      len(namespaces),
	}

	s.logger.Info("Successfully returned namespaces", zap.Int("count", response.Count))
	return response, nil
}

// HealthCheck verifies the service health
func (s *NamespaceService) HealthCheck(ctx context.Context) error {
	s.logger.Debug("Performing namespace service health check")
	return s.k8sClient.HealthCheck(ctx)
}