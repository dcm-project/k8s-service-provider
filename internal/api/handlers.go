package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dcm/service-provider/internal/deploy"
	"github.com/dcm/service-provider/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for the deployment service
type Handler struct {
	deployService *deploy.DeploymentService
	logger        *zap.Logger
}

// NewHandler creates a new API handler
func NewHandler(deployService *deploy.DeploymentService, logger *zap.Logger) *Handler {
	return &Handler{
		deployService: deployService,
		logger:        logger,
	}
}

// CreateDeployment handles POST /deployments
func (h *Handler) CreateDeployment(c *gin.Context) {
	logger := h.logger.Named("api_handler").With(zap.String("endpoint", "create_deployment"))

	var req models.DeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "INVALID_REQUEST",
			Message:   "Invalid request format",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Set default namespace if not provided
	if req.Metadata.Namespace == "" {
		req.Metadata.Namespace = "default"
	}

	// Generate unique ID for the deployment
	deploymentID := uuid.New().String()

	// Parse and validate the spec based on kind
	if err := h.parseAndValidateSpec(&req); err != nil {
		logger.Error("Failed to validate spec", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "INVALID_SPEC",
			Message:   "Invalid deployment specification",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Create the deployment
	if err := h.deployService.CreateDeployment(c.Request.Context(), &req, deploymentID); err != nil {
		logger.Error("Failed to create deployment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:      "DEPLOYMENT_FAILED",
			Message:   "Failed to create deployment",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Return the created deployment
	response := models.DeploymentResponse{
		ID:       deploymentID,
		Kind:     req.Kind,
		Metadata: req.Metadata,
		Spec:     req.Spec,
		Status: models.DeploymentStatus{
			Phase: models.DeploymentPhasePending,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	logger.Info("Successfully created deployment", zap.String("deployment_id", deploymentID))
	c.JSON(http.StatusCreated, response)
}

// GetDeployment handles GET /deployments/{id}
func (h *Handler) GetDeployment(c *gin.Context) {
	logger := h.logger.Named("api_handler").With(zap.String("endpoint", "get_deployment"))

	deploymentID := c.Param("id")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "MISSING_ID",
			Message:   "Deployment ID is required",
			Timestamp: time.Now(),
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	deployment, err := h.deployService.GetDeploymentByID(c.Request.Context(), deploymentID, namespace)
	if err != nil {
		logger.Error("Failed to get deployment", zap.Error(err))
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:      "DEPLOYMENT_NOT_FOUND",
			Message:   "Deployment not found",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	logger.Info("Successfully retrieved deployment", zap.String("deployment_id", deploymentID))
	c.JSON(http.StatusOK, deployment)
}

// UpdateDeployment handles PUT /deployments/{id}
func (h *Handler) UpdateDeployment(c *gin.Context) {
	logger := h.logger.Named("api_handler").With(zap.String("endpoint", "update_deployment"))

	deploymentID := c.Param("id")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "MISSING_ID",
			Message:   "Deployment ID is required",
			Timestamp: time.Now(),
		})
		return
	}

	var req models.DeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "INVALID_REQUEST",
			Message:   "Invalid request format",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Set default namespace if not provided
	if req.Metadata.Namespace == "" {
		req.Metadata.Namespace = "default"
	}

	// Parse and validate the spec based on kind
	if err := h.parseAndValidateSpec(&req); err != nil {
		logger.Error("Failed to validate spec", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "INVALID_SPEC",
			Message:   "Invalid deployment specification",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Update the deployment
	if err := h.deployService.UpdateDeployment(c.Request.Context(), &req, deploymentID); err != nil {
		logger.Error("Failed to update deployment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:      "UPDATE_FAILED",
			Message:   "Failed to update deployment",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Return the updated deployment
	response := models.DeploymentResponse{
		ID:       deploymentID,
		Kind:     req.Kind,
		Metadata: req.Metadata,
		Spec:     req.Spec,
		Status: models.DeploymentStatus{
			Phase: models.DeploymentPhasePending,
		},
		CreatedAt: time.Now(), // In a real implementation, preserve original creation time
		UpdatedAt: time.Now(),
	}

	logger.Info("Successfully updated deployment", zap.String("deployment_id", deploymentID))
	c.JSON(http.StatusOK, response)
}

// DeleteDeployment handles DELETE /deployments/{id}
func (h *Handler) DeleteDeployment(c *gin.Context) {
	logger := h.logger.Named("api_handler").With(zap.String("endpoint", "delete_deployment"))

	deploymentID := c.Param("id")
	if deploymentID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "MISSING_ID",
			Message:   "Deployment ID is required",
			Timestamp: time.Now(),
		})
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	kind := c.Query("kind")
	if kind == "" {
		// Try to determine kind by looking up the deployment
		deployment, err := h.deployService.GetDeploymentByID(c.Request.Context(), deploymentID, namespace)
		if err != nil {
			logger.Error("Failed to get deployment for deletion", zap.Error(err))
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Code:      "DEPLOYMENT_NOT_FOUND",
				Message:   "Deployment not found",
				Details:   err.Error(),
				Timestamp: time.Now(),
			})
			return
		}
		kind = string(deployment.Kind)
	}

	deploymentKind := models.DeploymentKind(kind)
	if deploymentKind != models.DeploymentKindContainer && deploymentKind != models.DeploymentKindVM {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "INVALID_KIND",
			Message:   "Invalid deployment kind",
			Details:   "Kind must be 'container' or 'vm'",
			Timestamp: time.Now(),
		})
		return
	}

	// Delete the deployment
	if err := h.deployService.DeleteDeployment(c.Request.Context(), deploymentID, namespace, deploymentKind); err != nil {
		logger.Error("Failed to delete deployment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:      "DELETE_FAILED",
			Message:   "Failed to delete deployment",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	logger.Info("Successfully deleted deployment", zap.String("deployment_id", deploymentID))
	c.Status(http.StatusNoContent)
}

// ListDeployments handles GET /deployments
func (h *Handler) ListDeployments(c *gin.Context) {
	logger := h.logger.Named("api_handler").With(zap.String("endpoint", "list_deployments"))

	var req models.ListDeploymentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Error("Failed to bind query parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:      "INVALID_QUERY",
			Message:   "Invalid query parameters",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 20
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	response, err := h.deployService.ListDeployments(c.Request.Context(), &req)
	if err != nil {
		logger.Error("Failed to list deployments", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:      "LIST_FAILED",
			Message:   "Failed to list deployments",
			Details:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	logger.Info("Successfully listed deployments", zap.Int("count", len(response.Deployments)))
	c.JSON(http.StatusOK, response)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusOK, response)
}

// parseAndValidateSpec parses and validates the deployment specification
func (h *Handler) parseAndValidateSpec(req *models.DeploymentRequest) error {
	// Convert the spec interface{} to proper typed spec based on kind
	specBytes, err := json.Marshal(req.Spec)
	if err != nil {
		return err
	}

	switch req.Kind {
	case models.DeploymentKindContainer:
		var containerSpec models.ContainerSpec
		if err := json.Unmarshal(specBytes, &containerSpec); err != nil {
			return err
		}
		req.Spec = containerSpec
	case models.DeploymentKindVM:
		var vmSpec models.VMSpec
		if err := json.Unmarshal(specBytes, &vmSpec); err != nil {
			return err
		}
		req.Spec = vmSpec
	default:
		return NewValidationError("unsupported deployment kind")
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	message string
}

func (e *ValidationError) Error() string {
	return e.message
}

// NewValidationError creates a new validation error
func NewValidationError(message string) *ValidationError {
	return &ValidationError{message: message}
}