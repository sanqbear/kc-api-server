package ews

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"kc-api/internal/utils"
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
		r.Get("/attachment", h.GetAttachment)
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
		utils.RespondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS plugin is not configured",
		})
		return
	}

	// Parse query parameters
	mailbox := r.URL.Query().Get("mailbox")
	if mailbox == "" {
		utils.RespondJSON(w, http.StatusBadRequest, ErrorResponse{
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
			utils.RespondJSON(w, http.StatusBadRequest, ErrorResponse{
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
			utils.RespondJSON(w, http.StatusBadRequest, ErrorResponse{
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
		utils.RespondInternalError(w, r, err, "Failed to retrieve emails from Exchange server")
		return
	}

	// Return response
	utils.RespondJSON(w, http.StatusOK, response)
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
		utils.RespondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS plugin is not configured",
		})
		return
	}

	// Get email ID from query parameter
	itemID := r.URL.Query().Get("item_id")
	if itemID == "" {
		utils.RespondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Bad Request",
			Message: "item_id parameter is required",
		})
		return
	}

	// Get mailbox from query parameter
	mailbox := r.URL.Query().Get("mailbox")
	if mailbox == "" {
		utils.RespondJSON(w, http.StatusBadRequest, ErrorResponse{
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
		// Check if it's a not found error
		if strings.Contains(err.Error(), "no message found") {
			utils.RespondJSON(w, http.StatusNotFound, ErrorResponse{
				Error:   "Not Found",
				Message: "Email not found",
			})
			return
		}

		utils.RespondInternalError(w, r, err, "Failed to retrieve email details from Exchange server")
		return
	}

	// Transform inline images in HTML body to base64 data URLs
	if response.Email.BodyType == "HTML" && len(response.Email.Attachments) > 0 {
		response.Email.Body = h.transformInlineImagesToDataURL(ctx, response.Email.Body, response.Email.Attachments)
	}

	// Return response
	utils.RespondJSON(w, http.StatusOK, response)
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
		utils.RespondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS client is not configured",
		})
		return
	}

	utils.RespondJSON(w, http.StatusOK, HealthResponse{
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

// GetAttachment handles downloading an attachment
// @Summary      Get attachment content
// @Description  Retrieves the binary content of an email attachment
// @Tags         plugins/ews
// @Produce      octet-stream
// @Security     BearerAuth
// @Param        mailbox query string true "Email address"
// @Param        attachment_id query string true "EWS Attachment ID"
// @Success      200 {file} binary "Attachment content"
// @Failure      400 {object} ErrorResponse "Invalid request parameters"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Failure      503 {object} ErrorResponse "EWS service unavailable"
// @Router       /plugins/ews/attachment [get]
func (h *Handler) GetAttachment(w http.ResponseWriter, r *http.Request) {
	// Check if EWS is configured
	if h.ewsClient == nil {
		utils.RespondJSON(w, http.StatusServiceUnavailable, ErrorResponse{
			Error:   "Service Unavailable",
			Message: "EWS plugin is not configured",
		})
		return
	}

	// Get attachment_id from query parameter
	attachmentID := r.URL.Query().Get("attachment_id")
	if attachmentID == "" {
		utils.RespondJSON(w, http.StatusBadRequest, ErrorResponse{
			Error:   "Bad Request",
			Message: "attachment_id parameter is required",
		})
		return
	}

	// Execute EWS request
	ctx := context.Background()
	content, err := h.ewsClient.GetAttachment(ctx, attachmentID)
	if err != nil {
		utils.RespondInternalError(w, r, err, "Failed to retrieve attachment from Exchange server")
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", content.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", content.Name))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)

	// Write binary content
	_, err = w.Write(content.Content)
	if err != nil {
		// Log error but can't change response at this point
		utils.RespondInternalError(w, r, err, "Failed to write attachment content")
		return
	}
}

// transformInlineImagesToDataURL replaces cid: references with base64 data URLs
func (h *Handler) transformInlineImagesToDataURL(ctx context.Context, htmlBody string, attachments []AttachmentInfo) string {
	for _, att := range attachments {
		if att.ContentId != "" && att.IsInline {
			// Get attachment content
			content, err := h.ewsClient.GetAttachment(ctx, att.AttachmentId)
			if err != nil {
				// Skip failed attachments
				continue
			}

			// Build data URL
			dataURL := fmt.Sprintf("data:%s;base64,%s",
				content.ContentType,
				base64.StdEncoding.EncodeToString(content.Content))

			// ContentId normalization (remove <...> if present)
			contentId := strings.Trim(att.ContentId, "<>")

			// Replace cid: patterns
			// Pattern 1: Full content ID
			htmlBody = strings.ReplaceAll(htmlBody, "cid:"+contentId, dataURL)

			// Pattern 2: Content ID without domain (before @)
			if idx := strings.Index(contentId, "@"); idx != -1 {
				localPart := contentId[:idx]
				htmlBody = strings.ReplaceAll(htmlBody, "cid:"+localPart, dataURL)
			}
		}
	}
	return htmlBody
}
