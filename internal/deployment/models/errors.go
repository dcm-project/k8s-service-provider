package models

import "fmt"

// ErrDeploymentNotFound represents an error when a deployment is not found
type ErrDeploymentNotFound struct {
	ID        string
	Namespace string // Optional, empty if searched across all namespaces
}

func (e *ErrDeploymentNotFound) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("deployment with ID %s not found in namespace %s", e.ID, e.Namespace)
	}
	return fmt.Sprintf("deployment with ID %s not found", e.ID)
}

// NewErrDeploymentNotFound creates a new ErrDeploymentNotFound
func NewErrDeploymentNotFound(id string, namespace ...string) *ErrDeploymentNotFound {
	err := &ErrDeploymentNotFound{ID: id}
	if len(namespace) > 0 {
		err.Namespace = namespace[0]
	}
	return err
}

// ErrMultipleDeploymentsFound represents an error when multiple deployments exist with the same ID
type ErrMultipleDeploymentsFound struct {
	ID         string
	Count      int
	Namespaces []string // Optional list of namespaces where conflicts were found
}

func (e *ErrMultipleDeploymentsFound) Error() string {
	if len(e.Namespaces) > 0 {
		return fmt.Sprintf("multiple deployments found with ID %s across namespaces: %v", e.ID, e.Namespaces)
	}
	return fmt.Sprintf("multiple deployments found with ID %s (%d conflicts)", e.ID, e.Count)
}

// NewErrMultipleDeploymentsFound creates a new ErrMultipleDeploymentsFound
func NewErrMultipleDeploymentsFound(id string, count int, namespaces ...string) *ErrMultipleDeploymentsFound {
	return &ErrMultipleDeploymentsFound{
		ID:         id,
		Count:      count,
		Namespaces: namespaces,
	}
}

// ErrDeploymentAlreadyExists represents an error when trying to create a deployment with an existing ID
type ErrDeploymentAlreadyExists struct {
	ID                string
	ExistingNamespace string
	ExistingKind      DeploymentKind
}

func (e *ErrDeploymentAlreadyExists) Error() string {
	return fmt.Sprintf("deployment with ID %s already exists in namespace %s (kind: %s)",
		e.ID, e.ExistingNamespace, e.ExistingKind)
}

// NewErrDeploymentAlreadyExists creates a new ErrDeploymentAlreadyExists
func NewErrDeploymentAlreadyExists(id, namespace string, kind DeploymentKind) *ErrDeploymentAlreadyExists {
	return &ErrDeploymentAlreadyExists{
		ID:                id,
		ExistingNamespace: namespace,
		ExistingKind:      kind,
	}
}

// Helper functions for error type checking

// IsNotFoundError checks if an error is a deployment not found error
func IsNotFoundError(err error) bool {
	_, ok := err.(*ErrDeploymentNotFound)
	return ok
}

// IsMultipleFoundError checks if an error is a multiple deployments found error
func IsMultipleFoundError(err error) bool {
	_, ok := err.(*ErrMultipleDeploymentsFound)
	return ok
}

// IsConflictError checks if an error is a deployment conflict error (already exists or multiple found)
func IsConflictError(err error) bool {
	switch err.(type) {
	case *ErrDeploymentAlreadyExists, *ErrMultipleDeploymentsFound:
		return true
	default:
		return false
	}
}

// IsAlreadyExistsError checks if an error is a deployment already exists error
func IsAlreadyExistsError(err error) bool {
	_, ok := err.(*ErrDeploymentAlreadyExists)
	return ok
}