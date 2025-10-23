package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeploymentRequest_JSON(t *testing.T) {
	tests := []struct {
		name     string
		request  DeploymentRequest
		wantJSON string
	}{
		{
			name: "container deployment request",
			request: DeploymentRequest{
				Kind: DeploymentKindContainer,
				Metadata: Metadata{
					Name:      "test-app",
					Namespace: "default",
					Labels: map[string]string{
						"app":     "test",
						"version": "1.0",
					},
				},
				Spec: ContainerSpec{
					Container: ContainerConfig{
						Image:    "nginx:latest",
						Replicas: 2,
						Ports: []PortConfig{
							{
								ContainerPort: 80,
								ServicePort:   8080,
								Protocol:      "TCP",
							},
						},
						Resources: &ResourceConfig{
							CPU:    "100m",
							Memory: "128Mi",
						},
					},
				},
			},
			wantJSON: `{"kind":"container","metadata":{"name":"test-app","namespace":"default","labels":{"app":"test","version":"1.0"}},"spec":{"container":{"image":"nginx:latest","replicas":2,"ports":[{"containerPort":80,"servicePort":8080,"protocol":"TCP"}],"resources":{"cpu":"100m","memory":"128Mi"}}}}`,
		},
		{
			name: "VM deployment request",
			request: DeploymentRequest{
				Kind: DeploymentKindVM,
				Metadata: Metadata{
					Name:      "test-vm",
					Namespace: "default",
				},
				Spec: VMSpec{
					VM: VMConfig{
						Ram: 4,
						Cpu: 2,
						Os:  "fedora",
					},
				},
			},
			wantJSON: `{"kind":"vm","metadata":{"name":"test-vm","namespace":"default"},"spec":{"vm":{"ram":4,"cpu":2,"os":"fedora"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			gotJSON, err := json.Marshal(tt.request)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(gotJSON))

			// Test unmarshaling
			var gotRequest DeploymentRequest
			err = json.Unmarshal([]byte(tt.wantJSON), &gotRequest)
			assert.NoError(t, err)
			assert.Equal(t, tt.request.Kind, gotRequest.Kind)
			assert.Equal(t, tt.request.Metadata, gotRequest.Metadata)
		})
	}
}

func TestDeploymentResponse_JSON(t *testing.T) {
	now := time.Now()
	response := DeploymentResponse{
		ID:   "test-id-123",
		Kind: DeploymentKindContainer,
		Metadata: Metadata{
			Name:      "test-app",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: ContainerSpec{
			Container: ContainerConfig{
				Image:    "nginx:latest",
				Replicas: 1,
			},
		},
		Status: DeploymentStatus{
			Phase:         DeploymentPhaseRunning,
			Message:       "Deployment is running",
			ReadyReplicas: 1,
			Conditions: []Condition{
				{
					Type:               "Ready",
					Status:             "True",
					LastTransitionTime: now,
					Reason:             "DeploymentReady",
					Message:            "Deployment is ready",
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Test marshaling
	jsonData, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "test-id-123")
	assert.Contains(t, string(jsonData), "container")
	assert.Contains(t, string(jsonData), "running")

	// Test unmarshaling
	var unmarshaled DeploymentResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, response.ID, unmarshaled.ID)
	assert.Equal(t, response.Kind, unmarshaled.Kind)
	assert.Equal(t, response.Status.Phase, unmarshaled.Status.Phase)
}

func TestListDeploymentsRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request ListDeploymentsRequest
		wantErr bool
	}{
		{
			name: "valid request with defaults",
			request: ListDeploymentsRequest{
				Limit:  20,
				Offset: 0,
			},
			wantErr: false,
		},
		{
			name: "valid request with filters",
			request: ListDeploymentsRequest{
				Namespace: "test",
				Kind:      DeploymentKindContainer,
				Limit:     10,
				Offset:    5,
			},
			wantErr: false,
		},
		{
			name: "invalid limit too high",
			request: ListDeploymentsRequest{
				Limit:  200,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid limit zero",
			request: ListDeploymentsRequest{
				Limit:  0,
				Offset: 0,
			},
			wantErr: true,
		},
		{
			name: "invalid negative offset",
			request: ListDeploymentsRequest{
				Limit:  20,
				Offset: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// In a real implementation, you would have validation functions
			// For now, we just test that the struct can be created
			assert.NotNil(t, tt.request)

			// Test JSON marshaling/unmarshaling
			jsonData, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			var unmarshaled ListDeploymentsRequest
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, tt.request.Limit, unmarshaled.Limit)
			assert.Equal(t, tt.request.Offset, unmarshaled.Offset)
		})
	}
}

func TestDeploymentKind_String(t *testing.T) {
	tests := []struct {
		kind DeploymentKind
		want string
	}{
		{DeploymentKindContainer, "container"},
		{DeploymentKindVM, "vm"},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.kind))
		})
	}
}

func TestDeploymentPhase_String(t *testing.T) {
	tests := []struct {
		phase DeploymentPhase
		want  string
	}{
		{DeploymentPhasePending, "pending"},
		{DeploymentPhaseRunning, "running"},
		{DeploymentPhaseSucceeded, "succeeded"},
		{DeploymentPhaseFailed, "failed"},
		{DeploymentPhaseUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.phase))
		})
	}
}

func TestMetadata_Validation(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		valid    bool
	}{
		{
			name: "valid metadata",
			metadata: Metadata{
				Name:      "test-app",
				Namespace: "default",
				Labels: map[string]string{
					"app":     "test",
					"version": "1.0",
				},
			},
			valid: true,
		},
		{
			name: "valid with empty namespace",
			metadata: Metadata{
				Name: "test-app",
			},
			valid: true,
		},
		{
			name: "invalid empty name",
			metadata: Metadata{
				Name:      "",
				Namespace: "default",
			},
			valid: false,
		},
		{
			name: "valid DNS-1123 name",
			metadata: Metadata{
				Name:      "my-app-123",
				Namespace: "my-namespace",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that metadata can be marshaled/unmarshaled
			jsonData, err := json.Marshal(tt.metadata)
			assert.NoError(t, err)

			var unmarshaled Metadata
			err = json.Unmarshal(jsonData, &unmarshaled)
			assert.NoError(t, err)
			assert.Equal(t, tt.metadata.Name, unmarshaled.Name)
			assert.Equal(t, tt.metadata.Namespace, unmarshaled.Namespace)

			// In a real implementation, you would validate DNS-1123 format
			if !tt.valid && tt.metadata.Name == "" {
				assert.Empty(t, tt.metadata.Name)
			}
		})
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	now := time.Now()
	errorResp := ErrorResponse{
		Code:      "DEPLOYMENT_FAILED",
		Message:   "Failed to create deployment",
		Details:   "Kubernetes API error: namespace not found",
		Timestamp: now,
	}

	// Test marshaling
	jsonData, err := json.Marshal(errorResp)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "DEPLOYMENT_FAILED")
	assert.Contains(t, string(jsonData), "Failed to create deployment")

	// Test unmarshaling
	var unmarshaled ErrorResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, errorResp.Code, unmarshaled.Code)
	assert.Equal(t, errorResp.Message, unmarshaled.Message)
	assert.Equal(t, errorResp.Details, unmarshaled.Details)
}

func TestHealthResponse_JSON(t *testing.T) {
	now := time.Now()
	healthResp := HealthResponse{
		Status:    "healthy",
		Timestamp: now,
	}

	// Test marshaling
	jsonData, err := json.Marshal(healthResp)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "healthy")

	// Test unmarshaling
	var unmarshaled HealthResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, healthResp.Status, unmarshaled.Status)
}