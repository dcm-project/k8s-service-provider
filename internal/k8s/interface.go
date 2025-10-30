package k8s

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// ClientInterface defines the interface for Kubernetes client operations
type ClientInterface interface {
	// GetClientset returns the underlying Kubernetes clientset
	GetClientset() kubernetes.Interface

	// HealthCheck verifies that the Kubernetes client can connect to the cluster
	HealthCheck(ctx context.Context) error

	// GetNamespacesByLabels retrieves namespaces that match the provided label selectors
	GetNamespacesByLabels(ctx context.Context, labelSelectors map[string]string) ([]NamespaceInfo, error)
}

// NamespaceInfo represents basic namespace information
type NamespaceInfo struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}
