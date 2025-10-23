package models

import "time"

// LabelSelectors represents the request body for filtering namespaces
type LabelSelectors struct {
	Labels map[string]string `json:"labels" validate:"required"`
}

// Namespace represents a Kubernetes namespace with its labels
type Namespace struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

// NamespaceResponse represents the response containing matching namespaces
type NamespaceResponse struct {
	Namespaces []Namespace `json:"namespaces"`
	Count      int         `json:"count"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}