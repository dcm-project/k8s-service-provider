package services

import (
	"context"
	"fmt"

	"github.com/dcm/k8s-service-provider/internal/deployment/models"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

// VMService handles virtual machine deployment operations using KubeVirt
type VMService struct {
	k8sClient      kubernetes.Interface
	kubevirtClient kubecli.KubevirtClient
	logger         *zap.Logger
}

// NewVMService creates a new VM service instance
func NewVMService(k8sClient kubernetes.Interface, logger *zap.Logger) *VMService {
	// Create KubeVirt client using default config
	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(kubecli.DefaultClientConfig(&pflag.FlagSet{}))
	if err != nil {
		logger.Fatal("Failed to create KubeVirt client", zap.Error(err))
	}

	return &VMService{
		k8sClient:      k8sClient,
		kubevirtClient: virtClient,
		logger:         logger,
	}
}

// CreateVM creates a new virtual machine deployment using KubeVirt
func (v *VMService) CreateVM(ctx context.Context, req *models.DeploymentRequest, id string) error {
	logger := v.logger.Named("vm_service").With(zap.String("deployment_id", id))
	logger.Info("Starting VM deployment")

	vmSpec, ok := req.Spec.(models.VMSpec)
	if !ok {
		return fmt.Errorf("invalid VM spec format")
	}

	namespace := req.Metadata.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Create namespace if it doesn't exist
	if err := v.ensureNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create the VirtualMachine object
	memory := resource.MustParse(fmt.Sprintf("%dGi", vmSpec.VM.Ram))
	virtualMachine := &kubevirtv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", req.Metadata.Name),
			Namespace:    namespace,
			Labels: map[string]string{
				"app-id": id,
			},
		},
		Spec: kubevirtv1.VirtualMachineSpec{
			RunStrategy: &[]kubevirtv1.VirtualMachineRunStrategy{kubevirtv1.RunStrategyRerunOnFailure}[0],
			Template: &kubevirtv1.VirtualMachineInstanceTemplateSpec{
				Spec: kubevirtv1.VirtualMachineInstanceSpec{
					Architecture: "amd64",
					Domain: kubevirtv1.DomainSpec{
						CPU: &kubevirtv1.CPU{
							Cores: uint32(vmSpec.VM.Cpu),
						},
						Memory: &kubevirtv1.Memory{
							Guest: &memory,
						},
						Devices: kubevirtv1.Devices{
							Disks: []kubevirtv1.Disk{
								{
									Name:      fmt.Sprintf("%s-disk", req.Metadata.Name),
									BootOrder: &[]uint{1}[0],
									DiskDevice: kubevirtv1.DiskDevice{
										Disk: &kubevirtv1.DiskTarget{
											Bus: kubevirtv1.DiskBusVirtio,
										},
									},
								},
								{
									Name:      "cloudinitdisk",
									BootOrder: &[]uint{2}[0],
									DiskDevice: kubevirtv1.DiskDevice{
										Disk: &kubevirtv1.DiskTarget{
											Bus: kubevirtv1.DiskBusVirtio,
										},
									},
								},
							},
							Interfaces: []kubevirtv1.Interface{
								{
									Name: "myvmnic",
									InterfaceBindingMethod: kubevirtv1.InterfaceBindingMethod{
										Bridge: &kubevirtv1.InterfaceBridge{},
									},
								},
							},
							Rng: &kubevirtv1.Rng{},
						},
						Features: &kubevirtv1.Features{
							ACPI: kubevirtv1.FeatureState{},
							SMM: &kubevirtv1.FeatureState{
								Enabled: &[]bool{true}[0],
							},
						},
						Machine: &kubevirtv1.Machine{
							Type: "pc-q35-rhel9.4.0",
						},
					},
					Networks: []kubevirtv1.Network{
						{
							Name: "myvmnic",
							NetworkSource: kubevirtv1.NetworkSource{
								Pod: &kubevirtv1.PodNetwork{},
							},
						},
					},
					TerminationGracePeriodSeconds: &[]int64{180}[0],
					Volumes: []kubevirtv1.Volume{
						{
							Name: fmt.Sprintf("%s-disk", req.Metadata.Name),
							VolumeSource: kubevirtv1.VolumeSource{
								ContainerDisk: &kubevirtv1.ContainerDiskSource{
									Image: v.getOSImage(vmSpec.VM.Os),
								},
							},
						},
						{
							Name: "cloudinitdisk",
							VolumeSource: kubevirtv1.VolumeSource{
								CloudInitNoCloud: &kubevirtv1.CloudInitNoCloudSource{
									UserData: v.generateCloudInitUserData(req.Metadata.Name, &vmSpec.VM),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the VirtualMachine in the cluster
	_, err := v.kubevirtClient.VirtualMachine(namespace).Create(ctx, virtualMachine, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create VirtualMachine: %w", err)
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

	// Get VirtualMachine by label selector
	vms, err := v.kubevirtClient.VirtualMachine(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app-id=%s", id),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual machine: %w", err)
	}

	if len(vms.Items) == 0 {
		return nil, fmt.Errorf("virtual machine not found")
	}

	vm := vms.Items[0]

	// Convert VirtualMachine to our response model
	response := &models.DeploymentResponse{
		ID:   id,
		Kind: models.DeploymentKindVM,
		Metadata: models.Metadata{
			Name:      vm.Name,
			Namespace: vm.Namespace,
			Labels:    vm.Labels,
		},
		Status: models.DeploymentStatus{
			Phase: v.getVMPhase(&vm),
		},
		CreatedAt: vm.CreationTimestamp.Time,
		UpdatedAt: vm.CreationTimestamp.Time,
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

	// Delete VirtualMachines
	err := v.kubevirtClient.VirtualMachine(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app-id=%s", id),
	})
	if err != nil {
		return fmt.Errorf("failed to delete VirtualMachine: %w", err)
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

	vms, err := v.kubevirtClient.VirtualMachine(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual machines: %w", err)
	}

	var responses []models.DeploymentResponse
	for i, vm := range vms.Items {
		if i < offset {
			continue
		}
		if len(responses) >= limit {
			break
		}

		appID := vm.Labels["app-id"]
		if appID == "" {
			continue // Skip VMs without app-id label
		}

		response := models.DeploymentResponse{
			ID:   appID,
			Kind: models.DeploymentKindVM,
			Metadata: models.Metadata{
				Name:      vm.Name,
				Namespace: vm.Namespace,
				Labels:    vm.Labels,
			},
			Status: models.DeploymentStatus{
				Phase: v.getVMPhase(&vm),
			},
			CreatedAt: vm.CreationTimestamp.Time,
			UpdatedAt: vm.CreationTimestamp.Time,
		}
		responses = append(responses, response)
	}

	logger.Info("Successfully listed VM deployments", zap.Int("count", len(responses)))
	return responses, nil
}

// ensureNamespace creates namespace if it doesn't exist
func (v *VMService) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := v.k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = v.k8sClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}
	return nil
}

// getOSImage returns the container image for the specified OS
func (v *VMService) getOSImage(os string) string {
	images := map[string]string{
		"fedora": "quay.io/containerdisks/fedora:latest",
		"ubuntu": "quay.io/containerdisks/ubuntu:latest",
		"centos": "quay.io/containerdisks/centos:latest",
		"rhel":   "quay.io/containerdisks/rhel:latest",
	}

	if image, exists := images[os]; exists {
		return image
	}
	// Default to fedora if OS not found
	return "quay.io/containerdisks/fedora:latest"
}

// generateCloudInitUserData generates cloud-init user data for the VM
func (v *VMService) generateCloudInitUserData(appName string, vm *models.VMConfig) string {
	return fmt.Sprintf(`#cloud-config
user: %s
password: auto-generated-pass
chpasswd: { expire: False }
hostname: %s
`, vm.Os, appName)
}

// getVMPhase converts KubeVirt VM status to our deployment phase
func (v *VMService) getVMPhase(vm *kubevirtv1.VirtualMachine) models.DeploymentPhase {
	if vm.Status.Ready {
		return models.DeploymentPhaseRunning
	}

	for _, condition := range vm.Status.Conditions {
		if condition.Type == kubevirtv1.VirtualMachineReady {
			if condition.Status == corev1.ConditionTrue {
				return models.DeploymentPhaseRunning
			}
		}
		if condition.Type == kubevirtv1.VirtualMachineFailure {
			if condition.Status == corev1.ConditionTrue {
				return models.DeploymentPhaseFailed
			}
		}
	}

	return models.DeploymentPhasePending
}