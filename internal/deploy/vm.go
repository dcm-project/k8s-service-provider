package deploy

import (
	"context"
	"fmt"

	"github.com/dcm/service-provider/internal/models"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// VMService handles virtual machine deployment operations
type VMService struct {
	client kubernetes.Interface
	logger *zap.Logger
}

// NewVMService creates a new VM service instance
func NewVMService(client kubernetes.Interface, logger *zap.Logger) *VMService {
	return &VMService{
		client: client,
		logger: logger,
	}
}

// CreateVM creates a new virtual machine deployment
func (v *VMService) CreateVM(ctx context.Context, req *models.DeploymentRequest, id string) error {
	logger := v.logger.Named("vm_service").With(zap.String("deployment_id", id))
	logger.Info("Starting VM deployment")

	spec, ok := req.Spec.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid VM spec format")
	}

	vmSpec, err := v.parseVMSpec(spec)
	if err != nil {
		return fmt.Errorf("failed to parse VM spec: %w", err)
	}

	namespace := req.Metadata.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Create namespace if it doesn't exist
	if err := v.ensureNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create SSH secret if SSH key is provided
	if vmSpec.VM.SSHKey != "" {
		if err := v.createSSHSecret(ctx, req.Metadata.Name, namespace, vmSpec.VM.SSHKey, id); err != nil {
			return fmt.Errorf("failed to create SSH secret: %w", err)
		}
	}

	// For now, we'll create a placeholder ConfigMap to represent the VM
	// In a real implementation, this would create KubeVirt VirtualMachine resources
	if err := v.createVMConfigMap(ctx, req.Metadata.Name, namespace, vmSpec, req.Metadata.Labels, id); err != nil {
		return fmt.Errorf("failed to create VM representation: %w", err)
	}

	logger.Info("Successfully created VM deployment")
	return nil
}

// GetVM retrieves VM deployment information
func (v *VMService) GetVM(ctx context.Context, id, namespace string) (*models.DeploymentResponse, error) {
	logger := v.logger.Named("vm_service").With(zap.String("deployment_id", id))

	if namespace == "" {
		namespace = "default"
	}

	// Get VM ConfigMap by label selector
	configMaps, err := v.client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app-id=%s,deployment-type=vm", id),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine: %w", err)
	}

	if len(configMaps.Items) == 0 {
		return nil, fmt.Errorf("virtual machine not found")
	}

	configMap := configMaps.Items[0]

	// Convert ConfigMap to our response model
	response := &models.DeploymentResponse{
		ID:   id,
		Kind: models.DeploymentKindVM,
		Metadata: models.Metadata{
			Name:      configMap.Labels["vm-name"],
			Namespace: configMap.Namespace,
			Labels:    configMap.Labels,
		},
		Status: models.DeploymentStatus{
			Phase: models.DeploymentPhaseRunning, // Simplified for demo
		},
		CreatedAt: configMap.CreationTimestamp.Time,
		UpdatedAt: configMap.CreationTimestamp.Time,
	}

	logger.Info("Successfully retrieved VM deployment")
	return response, nil
}

// UpdateVM updates an existing VM deployment
func (v *VMService) UpdateVM(ctx context.Context, req *models.DeploymentRequest, id string) error {
	logger := v.logger.Named("vm_service").With(zap.String("deployment_id", id))
	logger.Info("Updating VM deployment")

	namespace := req.Metadata.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// For simplicity, we'll delete and recreate the VM
	if err := v.DeleteVM(ctx, id, namespace); err != nil {
		logger.Warn("Failed to delete existing VM during update", zap.Error(err))
	}

	return v.CreateVM(ctx, req, id)
}

// DeleteVM deletes a virtual machine deployment
func (v *VMService) DeleteVM(ctx context.Context, id, namespace string) error {
	logger := v.logger.Named("vm_service").With(zap.String("deployment_id", id))
	logger.Info("Deleting VM deployment")

	if namespace == "" {
		namespace = "default"
	}

	// Delete ConfigMaps representing VMs
	configMaps, err := v.client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app-id=%s,deployment-type=vm", id),
	})
	if err != nil {
		logger.Error("Failed to list VM ConfigMaps", zap.Error(err))
		return fmt.Errorf("failed to list virtual machines: %w", err)
	}

	for _, configMap := range configMaps.Items {
		err = v.client.CoreV1().ConfigMaps(namespace).Delete(ctx, configMap.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Warn("Failed to delete VM ConfigMap", zap.String("configmap", configMap.Name), zap.Error(err))
		}
	}

	// Delete SSH secrets
	secrets, err := v.client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app-id=%s,secret-type=ssh", id),
	})
	if err != nil {
		logger.Warn("Failed to list SSH secrets", zap.Error(err))
	} else {
		for _, secret := range secrets.Items {
			err = v.client.CoreV1().Secrets(namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
			if err != nil {
				logger.Warn("Failed to delete SSH secret", zap.String("secret", secret.Name), zap.Error(err))
			}
		}
	}

	logger.Info("Successfully deleted VM deployment")
	return nil
}

// ListVMs lists all VM deployments
func (v *VMService) ListVMs(ctx context.Context, namespace string, limit, offset int) ([]models.DeploymentResponse, error) {
	logger := v.logger.Named("vm_service")

	if namespace == "" {
		namespace = "default"
	}

	configMaps, err := v.client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "deployment-type=vm",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual machines: %w", err)
	}

	var responses []models.DeploymentResponse
	for i, configMap := range configMaps.Items {
		if i < offset {
			continue
		}
		if len(responses) >= limit {
			break
		}

		response := models.DeploymentResponse{
			ID:   configMap.Labels["app-id"],
			Kind: models.DeploymentKindVM,
			Metadata: models.Metadata{
				Name:      configMap.Labels["vm-name"],
				Namespace: configMap.Namespace,
				Labels:    configMap.Labels,
			},
			Status: models.DeploymentStatus{
				Phase: models.DeploymentPhaseRunning, // Simplified for demo
			},
			CreatedAt: configMap.CreationTimestamp.Time,
			UpdatedAt: configMap.CreationTimestamp.Time,
		}
		responses = append(responses, response)
	}

	logger.Info("Successfully listed VM deployments", zap.Int("count", len(responses)))
	return responses, nil
}

// parseVMSpec parses the VM specification from the request
func (v *VMService) parseVMSpec(spec map[string]interface{}) (*models.VMSpec, error) {
	vmData, ok := spec["vm"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid VM specification")
	}

	image, ok := vmData["image"].(string)
	if !ok {
		return nil, fmt.Errorf("image is required")
	}

	cpu, ok := vmData["cpu"].(float64)
	if !ok {
		return nil, fmt.Errorf("cpu is required")
	}

	memory, ok := vmData["memory"].(string)
	if !ok {
		return nil, fmt.Errorf("memory is required")
	}

	vmSpec := &models.VMSpec{
		VM: models.VMConfig{
			Image:  image,
			CPU:    int(cpu),
			Memory: memory,
		},
	}

	if disk, ok := vmData["disk"].(string); ok {
		vmSpec.VM.Disk = disk
	}

	if sshKey, ok := vmData["sshKey"].(string); ok {
		vmSpec.VM.SSHKey = sshKey
	}

	if cloudInit, ok := vmData["cloudInit"].(string); ok {
		vmSpec.VM.CloudInit = cloudInit
	}

	return vmSpec, nil
}

// ensureNamespace creates namespace if it doesn't exist
func (v *VMService) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := v.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = v.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}
	return nil
}

// createSSHSecret creates a secret for SSH access
func (v *VMService) createSSHSecret(ctx context.Context, name, namespace, sshKey, id string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-ssh-%s", name, id[:8]),
			Labels: map[string]string{
				"app-id":      id,
				"secret-type": "ssh",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key": []byte(sshKey),
		},
	}

	_, err := v.client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

// createVMConfigMap creates a ConfigMap to represent the VM (placeholder for actual KubeVirt VM)
func (v *VMService) createVMConfigMap(ctx context.Context, name, namespace string, spec *models.VMSpec, labels map[string]string, id string) error {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app-id"] = id
	labels["vm-name"] = name
	labels["deployment-type"] = "vm"

	// Store VM specification in ConfigMap data
	vmData := map[string]string{
		"image":  spec.VM.Image,
		"cpu":    fmt.Sprintf("%d", spec.VM.CPU),
		"memory": spec.VM.Memory,
	}

	if spec.VM.Disk != "" {
		vmData["disk"] = spec.VM.Disk
	}

	if spec.VM.CloudInit != "" {
		vmData["cloudInit"] = spec.VM.CloudInit
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vm-%s", name, id[:8]),
			Namespace: namespace,
			Labels:    labels,
		},
		Data: vmData,
	}

	_, err := v.client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	return err
}