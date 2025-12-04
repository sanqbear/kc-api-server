package ews

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Handler handles Exchange Web Services related HTTP requests
type Handler struct {
	ewsClient *Client
}

// NewHandler creates a new EWS handler
func NewHandler(ewsClient *Client) *Handler {
	return &Handler{
		ewsClient: ewsClient,
	}
}

// RegisterRoutes registers the EWS plugin routes under /plugins/ews
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/plugins/ews", func(r chi.Router) {
		r.Get("/health", h.HealthCheck)
		r.Get("/emails", h.ListEmails)
		r.Get("/email", h.GetEmailDetail)
	})
}

// ListEmails handles retrieving a list of emails
// @Summary      List emails from Exchange mailbox
// @Description  Retrieve a list of emails from a specified Exchange mailbox and folder. Requires authentication.
// @Tags         plugins/ews
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        mailbox query string true "Email address of the mailbox to access"
// @Param        folder query string false "Folder name (inbox, sent, drafts, etc.)" default(inbox)
// @Param        limit query int false "Number of emails to retrieve (max 100)" default(50)
// @Param        offset query int false "Pagination offset" default(0)
// @Success      200 {object} ListEmailsResponse "Emails retrieved successfully"
// @Failure      400 {object} ErrorResponse "Invalid request parameters"
// @Failure      401 {object} ErrorResponse "Unauthorized - Invalid or missing token"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Failure      503 {object} ErrorResponse "EWS service unavailable"
// @Router       /plugins/ews/emails [get]
func (h *Handler) ListEmails(w http.ResponseWriter, r *http.Request) {
	// Check if EWS is configured
	if h.ewsClient == nil {
		respondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS plugin is not configured",
		})
		return
	}

	// Parse query parameters
	mailbox := r.URL.Query().Get("mailbox")
	if mailbox == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Bad Request",
			Message: "mailbox parameter is required",
		})
		return
	}

	folderName := r.URL.Query().Get("folder")
	if folderName == "" {
		folderName = "inbox"
	}

	// Parse limit
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error:   "Bad Request",
				Message: "invalid limit parameter",
			})
			return
		}
		limit = parsedLimit
	}

	// Parse offset
	offsetStr := r.URL.Query().Get("offset")
	offset := 0 // Default
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, ErrorResponse{
				Error:   "Bad Request",
				Message: "invalid offset parameter",
			})
			return
		}
		offset = parsedOffset
	}

	// Build request
	req := ListEmailsRequest{
		Mailbox:    mailbox,
		FolderName: folderName,
		Limit:      limit,
		Offset:     offset,
	}

	// Execute EWS request
	ctx := context.Background()
	response, err := h.ewsClient.ListEmails(ctx, req)
	if err != nil {
		log.Printf("Error listing emails: %v", err)
		respondJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to retrieve emails from Exchange server",
		})
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, response)
}

// GetEmailDetail handles retrieving full email details
// @Summary      Get email details by ID
// @Description  Retrieve full details of a specific email including body content and conversation thread. Requires authentication.
// @Tags         plugins/ews
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        item_id query string true "Email Item ID (Exchange item ID)"
// @Param        mailbox query string true "Email address of the mailbox to access"
// @Success      200 {object} GetEmailDetailResponse "Email details retrieved successfully"
// @Failure      400 {object} ErrorResponse "Invalid request parameters"
// @Failure      401 {object} ErrorResponse "Unauthorized - Invalid or missing token"
// @Failure      404 {object} ErrorResponse "Email not found"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Failure      503 {object} ErrorResponse "EWS service unavailable"
// @Router       /plugins/ews/email [get]
func (h *Handler) GetEmailDetail(w http.ResponseWriter, r *http.Request) {
	// Check if EWS is configured
	if h.ewsClient == nil {
		respondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS plugin is not configured",
		})
		return
	}

	// Get email ID from query parameter
	itemID := r.URL.Query().Get("item_id")
	if itemID == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Bad Request",
			Message: "item_id parameter is required",
		})
		return
	}

	// Get mailbox from query parameter
	mailbox := r.URL.Query().Get("mailbox")
	if mailbox == "" {
		respondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Bad Request",
			Message: "mailbox parameter is required",
		})
		return
	}

	// Build request
	req := GetEmailDetailRequest{
		Mailbox: mailbox,
		ItemID:  itemID,
	}

	// Execute EWS request
	ctx := context.Background()
	response, err := h.ewsClient.GetEmailDetail(ctx, req)
	if err != nil {
		log.Printf("Error getting email detail: %v", err)

		// Check if it's a not found error
		if strings.Contains(err.Error(), "no message found") {
			respondJSON(w, http.StatusNotFound, ErrorResponse{
				Error:   "Not Found",
				Message: "Email not found",
			})
			return
		}

		respondJSON(w, http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to retrieve email details from Exchange server",
		})
		return
	}

	// Return response
	respondJSON(w, http.StatusOK, response)
}

// HealthCheck returns the health status of the EWS integration
// @Summary      Check EWS connection health
// @Description  Check if the Exchange Web Services connection is properly configured and accessible
// @Tags         plugins/ews
// @Accept       json
// @Produce      json
// @Success      200 {object} HealthResponse "EWS connection is healthy"
// @Failure      503 {object} ErrorResponse "EWS service unavailable"
// @Router       /plugins/ews/health [get]
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if h.ewsClient == nil {
		respondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS client is not configured",
		})
		return
	}

	respondJSON(w, http.StatusOK, HealthResponse{
		Status:  "healthy",
		Message: "EWS client is configured and ready",
	})
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Bad Request"`
	Message string `json:"message" example:"Invalid request body"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status" example:"healthy"`
	Message string `json:"message" example:"EWS client is configured and ready"`
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
