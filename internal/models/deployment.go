package models

import (
	"time"
)

// DeploymentKind represents the type of deployment
type DeploymentKind string

const (
	DeploymentKindContainer DeploymentKind = "container"
	DeploymentKindVM        DeploymentKind = "vm"
)

// DeploymentRequest represents the request payload for creating/updating deployments
type DeploymentRequest struct {
	Kind     DeploymentKind `json:"kind" binding:"required,oneof=container vm"`
	Metadata Metadata       `json:"metadata" binding:"required"`
	Spec     interface{}    `json:"spec" binding:"required"`
}

// DeploymentResponse represents the response payload for deployments
type DeploymentResponse struct {
	ID        string            `json:"id"`
	Kind      DeploymentKind    `json:"kind"`
	Metadata  Metadata          `json:"metadata"`
	Spec      interface{}       `json:"spec"`
	Status    DeploymentStatus  `json:"status"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// Metadata represents common metadata for deployments
type Metadata struct {
	Name      string            `json:"name" binding:"required,max=63,min=1"`
	Namespace string            `json:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ContainerSpec represents the specification for container deployments
type ContainerSpec struct {
	Container ContainerConfig `json:"container" binding:"required"`
}

// ContainerConfig represents container configuration
type ContainerConfig struct {
	Image       string                `json:"image" binding:"required"`
	Replicas    *int                  `json:"replicas,omitempty"`
	Ports       []PortConfig          `json:"ports,omitempty"`
	Resources   *ResourceConfig       `json:"resources,omitempty"`
	Environment []EnvironmentVariable `json:"environment,omitempty"`
}

// PortConfig represents port configuration
type PortConfig struct {
	ContainerPort int    `json:"containerPort" binding:"required,min=1,max=65535"`
	ServicePort   int    `json:"servicePort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// ResourceConfig represents resource configuration
type ResourceConfig struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// VMSpec represents the specification for virtual machine deployments
type VMSpec struct {
	VM VMConfig `json:"vm" binding:"required"`
}

// VMConfig represents virtual machine configuration aligned with CatalogVm
type VMConfig struct {
	Ram int    `json:"ram" binding:"required,min=1,max=32"`
	Cpu int    `json:"cpu" binding:"required,min=1,max=32"`
	Os  string `json:"os" binding:"required"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	Phase         DeploymentPhase `json:"phase"`
	Message       string          `json:"message,omitempty"`
	ReadyReplicas int             `json:"readyReplicas,omitempty"`
	Conditions    []Condition     `json:"conditions,omitempty"`
}

// DeploymentPhase represents the phase of a deployment
type DeploymentPhase string

const (
	DeploymentPhasePending   DeploymentPhase = "pending"
	DeploymentPhaseRunning   DeploymentPhase = "running"
	DeploymentPhaseSucceeded DeploymentPhase = "succeeded"
	DeploymentPhaseFailed    DeploymentPhase = "failed"
	DeploymentPhaseUnknown   DeploymentPhase = "unknown"
)

// Condition represents a deployment condition
type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// ListDeploymentsRequest represents the request for listing deployments
type ListDeploymentsRequest struct {
	Namespace string         `form:"namespace"`
	Kind      DeploymentKind `form:"kind"`
	Limit     int            `form:"limit,default=20" binding:"min=1,max=100"`
	Offset    int            `form:"offset,default=0" binding:"min=0"`
}

// ListDeploymentsResponse represents the response for listing deployments
type ListDeploymentsResponse struct {
	Deployments []DeploymentResponse `json:"deployments"`
	Pagination  Pagination           `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	HasMore bool `json:"hasMore"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}