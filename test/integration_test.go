package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dcm/k8s-service-provider/internal/api"
	"github.com/dcm/k8s-service-provider/internal/deploy"
	"github.com/dcm/k8s-service-provider/internal/models"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"k8s.io/utils/ptr"
)

// IntegrationTestSuite defines the test suite for integration tests
type IntegrationTestSuite struct {
	suite.Suite
	router *httptest.Server
	logger *zap.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *IntegrationTestSuite) SetupSuite() {
	// Initialize logger
	suite.logger = zap.NewNop()

	// Create mock deployment service
	// In a real integration test, you might use test containers or in-memory implementations
	mockDeployService := &MockDeploymentService{}

	// Setup router
	ginRouter := api.SetupRouter(mockDeployService, suite.logger)
	suite.router = httptest.NewServer(ginRouter)
}

// TearDownSuite runs once after all tests in the suite
func (suite *IntegrationTestSuite) TearDownSuite() {
	suite.router.Close()
}

// MockDeploymentService is a simple mock for integration testing
type MockDeploymentService struct {
	deployments map[string]*models.DeploymentResponse
}

// Verify that MockDeploymentService implements DeploymentServiceInterface
var _ deploy.DeploymentServiceInterface = (*MockDeploymentService)(nil)

func (m *MockDeploymentService) CreateDeployment(ctx context.Context, req *models.DeploymentRequest, id string) error {
	if m.deployments == nil {
		m.deployments = make(map[string]*models.DeploymentResponse)
	}

	m.deployments[id] = &models.DeploymentResponse{
		ID:       id,
		Kind:     req.Kind,
		Metadata: req.Metadata,
		Spec:     req.Spec,
		Status: models.DeploymentStatus{
			Phase: models.DeploymentPhaseRunning,
		},
	}
	return nil
}

func (m *MockDeploymentService) GetDeploymentByID(ctx context.Context, id, namespace string) (*models.DeploymentResponse, error) {
	if m.deployments == nil {
		return nil, fmt.Errorf("deployment not found")
	}

	deployment, exists := m.deployments[id]
	if !exists {
		return nil, fmt.Errorf("deployment not found")
	}
	return deployment, nil
}

func (m *MockDeploymentService) UpdateDeployment(ctx context.Context, req *models.DeploymentRequest, id string) error {
	if m.deployments == nil {
		return fmt.Errorf("deployment not found")
	}

	if _, exists := m.deployments[id]; !exists {
		return fmt.Errorf("deployment not found")
	}

	m.deployments[id].Spec = req.Spec
	m.deployments[id].Metadata = req.Metadata
	return nil
}

func (m *MockDeploymentService) DeleteDeployment(ctx context.Context, id, namespace string, kind models.DeploymentKind) error {
	if m.deployments == nil {
		return fmt.Errorf("deployment not found")
	}

	if _, exists := m.deployments[id]; !exists {
		return fmt.Errorf("deployment not found")
	}

	delete(m.deployments, id)
	return nil
}

func (m *MockDeploymentService) ListDeployments(ctx context.Context, req *models.ListDeploymentsRequest) (*models.ListDeploymentsResponse, error) {
	if m.deployments == nil {
		return &models.ListDeploymentsResponse{
			Deployments: []models.DeploymentResponse{},
			Pagination: models.Pagination{
				Limit:   req.Limit,
				Offset:  req.Offset,
				Total:   0,
				HasMore: false,
			},
		}, nil
	}

	var deployments []models.DeploymentResponse
	for _, deployment := range m.deployments {
		// Apply filters
		if req.Kind != "" && deployment.Kind != req.Kind {
			continue
		}
		if req.Namespace != "" && deployment.Metadata.Namespace != req.Namespace {
			continue
		}
		deployments = append(deployments, *deployment)
	}

	// Apply pagination
	total := len(deployments)
	start := req.Offset
	end := start + req.Limit

	if start >= total {
		deployments = []models.DeploymentResponse{}
	} else {
		if end > total {
			end = total
		}
		deployments = deployments[start:end]
	}

	return &models.ListDeploymentsResponse{
		Deployments: deployments,
		Pagination: models.Pagination{
			Limit:   req.Limit,
			Offset:  req.Offset,
			Total:   total,
			HasMore: req.Offset+req.Limit < total,
		},
	}, nil
}

func (suite *IntegrationTestSuite) TestHealthCheck() {
	resp, err := http.Get(suite.router.URL + "/api/v1/health")
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var healthResp models.HealthResponse
	err = json.NewDecoder(resp.Body).Decode(&healthResp)
	suite.NoError(err)
	suite.Equal("healthy", healthResp.Status)
}

func (suite *IntegrationTestSuite) TestContainerDeploymentLifecycle() {
	// Test creating a container deployment
	createReq := models.DeploymentRequest{
		Kind: models.DeploymentKindContainer,
		Metadata: models.Metadata{
			Name:      "test-nginx",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: models.ContainerSpec{
			Container: models.ContainerConfig{
				Image:    "nginx:latest",
				Replicas: ptr.To(2),
				Ports: []models.PortConfig{
					{
						ContainerPort: 80,
						ServicePort:   8080,
					},
				},
			},
		},
	}

	// Create deployment
	createBody, _ := json.Marshal(createReq)
	resp, err := http.Post(suite.router.URL+"/api/v1/deployments", "application/json", bytes.NewBuffer(createBody))
	suite.NoError(err)
	suite.Equal(http.StatusCreated, resp.StatusCode)

	var createResp models.DeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	suite.NoError(err)
	suite.Equal(models.DeploymentKindContainer, createResp.Kind)
	suite.Equal("test-nginx", createResp.Metadata.Name)
	deploymentID := createResp.ID

	// Get deployment
	resp, err = http.Get(suite.router.URL + "/api/v1/deployments/" + deploymentID)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var getResp models.DeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	suite.NoError(err)
	suite.Equal(deploymentID, getResp.ID)
	suite.Equal(models.DeploymentKindContainer, getResp.Kind)

	// Update deployment
	updateReq := createReq
	containerSpec := updateReq.Spec.(models.ContainerSpec)
	containerSpec.Container.Replicas = ptr.To(3)
	updateReq.Spec = containerSpec
	updateBody, _ := json.Marshal(updateReq)

	client := &http.Client{}
	req, _ := http.NewRequest("PUT", suite.router.URL+"/api/v1/deployments/"+deploymentID, bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	// List deployments
	resp, err = http.Get(suite.router.URL + "/api/v1/deployments")
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var listResp models.ListDeploymentsResponse
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	suite.NoError(err)
	suite.True(len(listResp.Deployments) > 0)

	// Delete deployment
	req, _ = http.NewRequest("DELETE", suite.router.URL+"/api/v1/deployments/"+deploymentID+"?kind=container", nil)
	resp, err = client.Do(req)
	suite.NoError(err)
	suite.Equal(http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	resp, err = http.Get(suite.router.URL + "/api/v1/deployments/" + deploymentID)
	suite.NoError(err)
	suite.Equal(http.StatusNotFound, resp.StatusCode)
}

func (suite *IntegrationTestSuite) TestVMDeploymentLifecycle() {
	// Test creating a VM deployment
	createReq := models.DeploymentRequest{
		Kind: models.DeploymentKindVM,
		Metadata: models.Metadata{
			Name:      "test-fedora-vm",
			Namespace: "default",
			Labels: map[string]string{
				"os": "fedora",
			},
		},
		Spec: models.VMSpec{
			VM: models.VMConfig{
				Ram: 4,
				Cpu: 2,
				Os:  "fedora",
			},
		},
	}

	// Create deployment
	createBody, _ := json.Marshal(createReq)
	resp, err := http.Post(suite.router.URL+"/api/v1/deployments", "application/json", bytes.NewBuffer(createBody))
	suite.NoError(err)
	suite.Equal(http.StatusCreated, resp.StatusCode)

	var createResp models.DeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&createResp)
	suite.NoError(err)
	suite.Equal(models.DeploymentKindVM, createResp.Kind)
	suite.Equal("test-fedora-vm", createResp.Metadata.Name)
	deploymentID := createResp.ID

	// Get deployment
	resp, err = http.Get(suite.router.URL + "/api/v1/deployments/" + deploymentID)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Delete deployment
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", suite.router.URL+"/api/v1/deployments/"+deploymentID+"?kind=vm", nil)
	resp, err = client.Do(req)
	suite.NoError(err)
	suite.Equal(http.StatusNoContent, resp.StatusCode)
}

func (suite *IntegrationTestSuite) TestErrorHandling() {
	// Test invalid JSON
	resp, err := http.Post(suite.router.URL+"/api/v1/deployments", "application/json", bytes.NewBuffer([]byte("invalid json")))
	suite.NoError(err)
	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	// Test invalid endpoint
	resp, err = http.Get(suite.router.URL + "/api/v1/invalid-endpoint")
	suite.NoError(err)
	suite.Equal(http.StatusNotFound, resp.StatusCode) // Router will return 404 for missing path

	// Test non-existent deployment
	resp, err = http.Get(suite.router.URL + "/api/v1/deployments/non-existent-id")
	suite.NoError(err)
	suite.Equal(http.StatusNotFound, resp.StatusCode)
}

func (suite *IntegrationTestSuite) TestListDeploymentsWithFilters() {
	// Create a container deployment
	containerReq := models.DeploymentRequest{
		Kind: models.DeploymentKindContainer,
		Metadata: models.Metadata{
			Name:      "test-container",
			Namespace: "test-namespace",
		},
		Spec: models.ContainerSpec{
			Container: models.ContainerConfig{
				Image: "nginx:latest",
			},
		},
	}

	containerBody, _ := json.Marshal(containerReq)
	resp, err := http.Post(suite.router.URL+"/api/v1/deployments", "application/json", bytes.NewBuffer(containerBody))
	suite.NoError(err)
	suite.Equal(http.StatusCreated, resp.StatusCode)

	// Create a VM deployment
	vmReq := models.DeploymentRequest{
		Kind: models.DeploymentKindVM,
		Metadata: models.Metadata{
			Name:      "test-vm",
			Namespace: "test-namespace",
		},
		Spec: models.VMSpec{
			VM: models.VMConfig{
				Ram: 2,
				Cpu: 1,
				Os:  "fedora",
			},
		},
	}

	vmBody, _ := json.Marshal(vmReq)
	resp, err = http.Post(suite.router.URL+"/api/v1/deployments", "application/json", bytes.NewBuffer(vmBody))
	suite.NoError(err)
	suite.Equal(http.StatusCreated, resp.StatusCode)

	// Test filtering by kind
	resp, err = http.Get(suite.router.URL + "/api/v1/deployments?kind=container&namespace=test-namespace")
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	var listResp models.ListDeploymentsResponse
	err = json.NewDecoder(resp.Body).Decode(&listResp)
	suite.NoError(err)

	// Should only return container deployments
	for _, deployment := range listResp.Deployments {
		suite.Equal(models.DeploymentKindContainer, deployment.Kind)
	}
}

// TestIntegrationSuite runs the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// TestMain allows running tests with setup/teardown
func TestMain(m *testing.M) {
	// Run tests
	m.Run()
}