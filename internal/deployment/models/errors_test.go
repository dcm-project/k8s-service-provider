package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrDeploymentNotFound(t *testing.T) {
	t.Run("error without namespace", func(t *testing.T) {
		err := NewErrDeploymentNotFound("test-id")
		assert.Equal(t, "deployment with ID test-id not found", err.Error())
		assert.True(t, IsNotFoundError(err))
		assert.False(t, IsConflictError(err))
		assert.False(t, IsMultipleFoundError(err))
		assert.False(t, IsAlreadyExistsError(err))
	})

	t.Run("error with namespace", func(t *testing.T) {
		err := NewErrDeploymentNotFound("test-id", "test-namespace")
		assert.Equal(t, "deployment with ID test-id not found in namespace test-namespace", err.Error())
		assert.True(t, IsNotFoundError(err))
	})
}

func TestErrMultipleDeploymentsFound(t *testing.T) {
	t.Run("error without namespaces", func(t *testing.T) {
		err := NewErrMultipleDeploymentsFound("test-id", 3)
		assert.Equal(t, "multiple deployments found with ID test-id (3 conflicts)", err.Error())
		assert.True(t, IsMultipleFoundError(err))
		assert.True(t, IsConflictError(err))
		assert.False(t, IsNotFoundError(err))
		assert.False(t, IsAlreadyExistsError(err))
	})

	t.Run("error with namespaces", func(t *testing.T) {
		err := NewErrMultipleDeploymentsFound("test-id", 2, "namespace1", "namespace2")
		assert.Equal(t, "multiple deployments found with ID test-id across namespaces: [namespace1 namespace2]", err.Error())
		assert.True(t, IsMultipleFoundError(err))
		assert.True(t, IsConflictError(err))
	})
}

func TestErrDeploymentAlreadyExists(t *testing.T) {
	err := NewErrDeploymentAlreadyExists("test-id", "test-namespace", DeploymentKindContainer)
	assert.Equal(t, "deployment with ID test-id already exists in namespace test-namespace (kind: container)", err.Error())
	assert.True(t, IsAlreadyExistsError(err))
	assert.True(t, IsConflictError(err))
	assert.False(t, IsNotFoundError(err))
	assert.False(t, IsMultipleFoundError(err))
}

func TestErrorTypeChecking(t *testing.T) {
	t.Run("nil error checks", func(t *testing.T) {
		assert.False(t, IsNotFoundError(nil))
		assert.False(t, IsMultipleFoundError(nil))
		assert.False(t, IsConflictError(nil))
		assert.False(t, IsAlreadyExistsError(nil))
	})

	t.Run("regular error checks", func(t *testing.T) {
		regularErr := assert.AnError
		assert.False(t, IsNotFoundError(regularErr))
		assert.False(t, IsMultipleFoundError(regularErr))
		assert.False(t, IsConflictError(regularErr))
		assert.False(t, IsAlreadyExistsError(regularErr))
	})
}
