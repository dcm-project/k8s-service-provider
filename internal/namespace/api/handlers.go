package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dcm-project/k8s-service-provider/internal/namespace/models"
	"github.com/dcm-project/k8s-service-provider/internal/namespace/services"
	"go.uber.org/zap"
)

// Handler contains dependencies for HTTP handlers
type Handler struct {
	namespaceService *services.NamespaceService
	logger           *zap.Logger
}

// NewHandler creates a new handler instance
func NewHandler(namespaceService *services.NamespaceService, logger *zap.Logger) *Handler {
	return &Handler{
		namespaceService: namespaceService,
		logger:           logger,
	}
}

// GetNamespacesByLabels handles POST /api/v1/namespaces requests
func (h *Handler) GetNamespacesByLabels(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Received request to get namespaces by labels")

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req models.LabelSelectors
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to decode request body", zap.Error(err))
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", "Failed to parse request body")
		return
	}

	// Validate request
	if req.Labels == nil || len(req.Labels) == 0 {
		h.logger.Error("Empty labels provided")
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation Error", "Labels cannot be empty")
		return
	}

	// Log the label selectors
	h.logger.Info("Processing label selectors", zap.Any("labels", req.Labels))

	// Get namespaces from service
	response, err := h.namespaceService.GetNamespacesByLabels(r.Context(), req.Labels)
	if err != nil {
		h.logger.Error("Failed to get namespaces from service", zap.Error(err))
		h.writeErrorResponse(w, http.StatusInternalServerError, "Kubernetes API Error", "Failed to fetch namespaces")
		return
	}

	// Write successful response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", zap.Error(err))
		return
	}

	h.logger.Info("Successfully returned namespaces", zap.Int("count", response.Count))
}

// HealthCheck handles GET /api/v1/health requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("Received health check request")

	w.Header().Set("Content-Type", "application/json")

	// Check service health
	err := h.namespaceService.HealthCheck(r.Context())

	response := models.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
	}

	if err != nil {
		h.logger.Error("Health check failed", zap.Error(err))
		response.Status = "unhealthy"
		response.Error = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health check response", zap.Error(err))
		return
	}
}

// writeErrorResponse writes a standardized error response
func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, errorType, message string) {
	response := models.ErrorResponse{
		Error:   errorType,
		Message: message,
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode error response", zap.Error(err))
	}
}

// NotFoundHandler handles 404 errors
func (h *Handler) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Warn("Endpoint not found", zap.String("path", r.URL.Path))
	w.Header().Set("Content-Type", "application/json")
	h.writeErrorResponse(w, http.StatusNotFound, "Not Found", "The requested endpoint does not exist")
}

// MethodNotAllowedHandler handles 405 errors
func (h *Handler) MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	h.logger.Warn("Method not allowed",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)
	w.Header().Set("Content-Type", "application/json")
	h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed", "The HTTP method is not allowed for this endpoint")
}