package aiqueue

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
)

// Handler handles HTTP requests for AI queue operations
type Handler struct {
	service Service
}

// NewHandler creates a new AI queue handler with the given service
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers AI queue routes on the given router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api/ai", func(r chi.Router) {
		r.Post("/summarize", h.Summarize)
		r.Post("/keywords", h.ExtractKeywords)
		r.Post("/normalize", h.NormalizeRequest)
		r.Get("/tasks/{id}", h.GetTaskStatus)
		r.Delete("/tasks/{id}", h.DeleteTask)
	})
}

// Summarize godoc
// @Summary      Submit text summarization task
// @Description  Submits a text summarization task to the AI worker queue and returns a task ID for tracking
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        request  body      SummarizeRequest  true  "Summarization request"
// @Success      202      {object}  SummarizeResponse
// @Failure      400      {object}  utils.ErrorResponse
// @Failure      500      {object}  utils.ErrorResponse
// @Security     BearerAuth
// @Router       /api/ai/summarize [post]
func (h *Handler) Summarize(w http.ResponseWriter, r *http.Request) {
	var req SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate input
	if req.Text == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Text is required")
		return
	}

	// Submit task
	taskID, err := h.service.Summarize(r.Context(), req.Text, req.MaxLength)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		if errors.Is(err, ErrClientNotInitialized) {
			utils.RespondError(w, r, http.StatusServiceUnavailable, "Service Unavailable", "AI worker queue is not available")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to submit summarization task")
		return
	}

	utils.RespondJSON(w, http.StatusAccepted, SummarizeResponse{
		TaskID:  taskID,
		Message: "Summarization task submitted successfully",
	})
}

// ExtractKeywords godoc
// @Summary      Submit keyword extraction task
// @Description  Submits a keyword extraction task to the AI worker queue and returns a task ID for tracking
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        request  body      KeywordsRequest  true  "Keyword extraction request"
// @Success      202      {object}  KeywordsResponse
// @Failure      400      {object}  utils.ErrorResponse
// @Failure      500      {object}  utils.ErrorResponse
// @Security     BearerAuth
// @Router       /api/ai/keywords [post]
func (h *Handler) ExtractKeywords(w http.ResponseWriter, r *http.Request) {
	var req KeywordsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate input
	if req.Text == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Text is required")
		return
	}

	// Submit task
	taskID, err := h.service.ExtractKeywords(r.Context(), req.Text, req.MaxKeywords)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		if errors.Is(err, ErrClientNotInitialized) {
			utils.RespondError(w, r, http.StatusServiceUnavailable, "Service Unavailable", "AI worker queue is not available")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to submit keyword extraction task")
		return
	}

	utils.RespondJSON(w, http.StatusAccepted, KeywordsResponse{
		TaskID:  taskID,
		Message: "Keyword extraction task submitted successfully",
	})
}

// NormalizeRequest godoc
// @Summary      Submit JSON normalization task
// @Description  Submits a JSON normalization task to the AI worker queue and returns a task ID for tracking. This normalizes unstructured text into a structured JSON format based on the provided schema.
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        request  body      NormalizeRequest  true  "Normalization request"
// @Success      202      {object}  NormalizeResponse
// @Failure      400      {object}  utils.ErrorResponse
// @Failure      500      {object}  utils.ErrorResponse
// @Security     BearerAuth
// @Router       /api/ai/normalize [post]
func (h *Handler) NormalizeRequest(w http.ResponseWriter, r *http.Request) {
	var req NormalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, r, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// Validate input
	if req.Request == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Request text is required")
		return
	}
	if req.Schema == nil || len(req.Schema) == 0 {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Schema is required")
		return
	}

	// Submit task
	taskID, err := h.service.NormalizeRequest(r.Context(), req.Request, req.Schema)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		if errors.Is(err, ErrClientNotInitialized) {
			utils.RespondError(w, r, http.StatusServiceUnavailable, "Service Unavailable", "AI worker queue is not available")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to submit normalization task")
		return
	}

	utils.RespondJSON(w, http.StatusAccepted, NormalizeResponse{
		TaskID:  taskID,
		Message: "Normalization task submitted successfully",
	})
}

// GetTaskStatus godoc
// @Summary      Get task status and result
// @Description  Retrieves the status and result of a previously submitted AI task. Returns task status (PENDING, STARTED, SUCCESS, FAILURE) and the result if available.
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Task ID"
// @Success      200  {object}  TaskStatusResponse
// @Failure      400  {object}  utils.ErrorResponse
// @Failure      404  {object}  utils.ErrorResponse
// @Failure      500  {object}  utils.ErrorResponse
// @Security     BearerAuth
// @Router       /api/ai/tasks/{id} [get]
func (h *Handler) GetTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Task ID is required")
		return
	}

	// Get task result
	result, err := h.service.GetTaskResult(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			utils.RespondError(w, r, http.StatusNotFound, "Not Found", "Task not found")
			return
		}
		utils.RespondInternalError(w, r, err, "Failed to get task status")
		return
	}

	// Convert to response format
	response := TaskStatusResponse{
		TaskID:      result.ID,
		Status:      result.Status,
		Result:      result.Result,
		Error:       result.Error,
		StartedAt:   result.StartedAt,
		CompletedAt: result.CompletedAt,
	}

	utils.RespondJSON(w, http.StatusOK, response)
}

// DeleteTask godoc
// @Summary      Delete task result
// @Description  Deletes the result of a completed task from the queue system. This is useful for cleanup after retrieving results.
// @Tags         ai
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Task ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  utils.ErrorResponse
// @Failure      500  {object}  utils.ErrorResponse
// @Security     BearerAuth
// @Router       /api/ai/tasks/{id} [delete]
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		utils.RespondError(w, r, http.StatusBadRequest, "Bad Request", "Task ID is required")
		return
	}

	// Delete task result
	if err := h.service.DeleteTaskResult(r.Context(), taskID); err != nil {
		utils.RespondInternalError(w, r, err, "Failed to delete task result")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Task result deleted successfully",
	})
}
