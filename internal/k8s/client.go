package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/dcm-project/k8s-service-provider/internal/config"
)

// Client wraps the Kubernetes client and provides shared functionality
type Client struct {
	clientset kubernetes.Interface
	logger    *zap.Logger
}

// NewClient creates a new shared Kubernetes client
func NewClient(cfg config.KubernetesConfig, logger *zap.Logger) (ClientInterface, error) {
	k8sConfig, err := getKubeConfig(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Client{
		clientset: clientset,
		logger:    logger,
	}, nil
}

// GetClientset returns the underlying Kubernetes clientset
func (c *Client) GetClientset() kubernetes.Interface {
	return c.clientset
}

// HealthCheck verifies that the Kubernetes client can connect to the cluster
func (c *Client) HealthCheck(ctx context.Context) error {
	c.logger.Debug("Performing Kubernetes health check")

	// Try to get server version as a simple health check
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		c.logger.Error("Kubernetes health check failed", zap.Error(err))
		return fmt.Errorf("kubernetes health check failed: %w", err)
	}

	c.logger.Debug("Kubernetes health check successful")
	return nil
}

// GetNamespacesByLabels retrieves namespaces that match the provided label selectors
func (c *Client) GetNamespacesByLabels(ctx context.Context, labelSelectors map[string]string) ([]NamespaceInfo, error) {
	c.logger.Info("Fetching namespaces by labels", zap.Any("labelSelectors", labelSelectors))

	// Convert label selectors to Kubernetes label selector
	selector := labels.Set(labelSelectors).AsSelector()

	// List namespaces with label selector
	namespaceList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		c.logger.Error("Failed to list namespaces", zap.Error(err))
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	// Convert to response format
	namespaces := make([]NamespaceInfo, 0, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		namespace := NamespaceInfo{
			Name:   ns.Name,
			Labels: ns.Labels,
		}
		// Ensure labels map is not nil
		if namespace.Labels == nil {
			namespace.Labels = make(map[string]string)
		}
		namespaces = append(namespaces, namespace)
	}

	c.logger.Info("Successfully retrieved namespaces", zap.Int("count", len(namespaces)))
	return namespaces, nil
}

// getKubeConfig returns the Kubernetes configuration based on the provided config
func getKubeConfig(cfg config.KubernetesConfig, logger *zap.Logger) (*rest.Config, error) {
	var k8sConfig *rest.Config
	var err error

	if cfg.InCluster {
		logger.Info("Using in-cluster Kubernetes configuration")
		k8sConfig, err = rest.InClusterConfig()
	} else {
		configPath := cfg.ConfigPath
		if configPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get user home directory: %w", err)
			}
			configPath = filepath.Join(home, ".kube", "config")
		}

		logger.Info("Using kubeconfig file", zap.String("path", configPath))
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", configPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
	}

	logger.Info("Successfully initialized Kubernetes configuration")
	return k8sConfig, nil
}
