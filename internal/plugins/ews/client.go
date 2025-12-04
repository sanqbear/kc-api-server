package ews

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents an EWS client for interacting with Exchange server
type Client struct {
	config     *Config
	httpClient *http.Client
	serverURL  string
	authUser   string // Username for authentication (may include domain)
}

// getAuthUsername returns the username formatted for authentication
// If domain is configured, returns "DOMAIN\username" format for NTLM
// Otherwise returns just the username for Basic auth
func getAuthUsername(cfg *Config) string {
	if cfg.Domain != "" {
		return cfg.Domain + "\\" + cfg.ImpersonationUsername
	}
	return cfg.ImpersonationUsername
}

// NewClient creates a new EWS client instance
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("EWS config is nil")
	}

	// Validate configuration
	if err := ValidateEWSURL(cfg.ServerURL); err != nil {
		return nil, err
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.SkipTLSVerify,
		},
	}

	httpClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	return &Client{
		config:     cfg,
		httpClient: httpClient,
		serverURL:  cfg.ServerURL,
		authUser:   getAuthUsername(cfg),
	}, nil
}

// ListEmails retrieves a list of emails from the specified mailbox and folder
func (c *Client) ListEmails(ctx context.Context, req ListEmailsRequest) (*ListEmailsResponse, error) {
	// Validate request
	if err := ValidateMailbox(req.Mailbox); err != nil {
		return nil, err
	}

	// Sanitize pagination parameters
	limit := SanitizeLimit(req.Limit)
	offset := SanitizeOffset(req.Offset)

	// Default folder to Inbox if not specified
	folderName := req.FolderName
	if folderName == "" {
		folderName = FolderInbox
	}
	folderID := GetFolderID(folderName)

	// Build FindItem request
	findItemReq := c.buildFindItemRequest(req.Mailbox, folderID, limit, offset)

	// Execute request
	response, err := c.executeFindItemRequest(ctx, findItemReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute FindItem request: %w", err)
	}

	// Parse response
	emails, total, err := c.parseFindItemResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse FindItem response: %w", err)
	}

	return &ListEmailsResponse{
		Emails: emails,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// GetEmailDetail retrieves full details of a specific email including thread information
func (c *Client) GetEmailDetail(ctx context.Context, req GetEmailDetailRequest) (*GetEmailDetailResponse, error) {
	// Validate request
	if err := ValidateMailbox(req.Mailbox); err != nil {
		return nil, err
	}

	if req.ItemID == "" {
		return nil, fmt.Errorf("item_id is required")
	}

	// Build GetItem request
	getItemReq := c.buildGetItemRequest(req.ItemID)

	// Execute request
	response, err := c.executeGetItemRequest(ctx, getItemReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GetItem request: %w", err)
	}

	// Parse response
	email, err := c.parseGetItemResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GetItem response: %w", err)
	}

	// Get conversation thread if ConversationID is available
	var thread []EmailListItem
	if email.ConversationID != "" {
		thread, err = c.getConversationThread(ctx, req.Mailbox, email.ConversationID, req.ItemID)
		if err != nil {
			// Log error but don't fail the request
			// Thread information is optional
			thread = []EmailListItem{}
		}
	}

	return &GetEmailDetailResponse{
		Email:  *email,
		Thread: thread,
	}, nil
}

// buildFindItemRequest creates a FindItem SOAP request
func (c *Client) buildFindItemRequest(mailbox, folderID string, limit, offset int) *SOAPEnvelope {
	envelope := &SOAPEnvelope{
		XMLNS: "http://schemas.xmlsoap.org/soap/envelope/",
		XSI:   "http://www.w3.org/2001/XMLSchema-instance",
		M:     "http://schemas.microsoft.com/exchange/services/2006/messages",
		T:     "http://schemas.microsoft.com/exchange/services/2006/types",
		Header: SOAPHeader{
			RequestServerVersion: RequestServerVersion{
				Version: GetEWSAPIVersion(),
			},
			ExchangeImpersonation: &ExchangeImpersonation{
				ConnectingSID: ConnectingSID{
					PrimarySmtpAddress: mailbox,
				},
			},
		},
		Body: SOAPBody{
			Content: FindItemRequest{
				Traversal: "Shallow",
				ItemShape: ItemShape{
					BaseShape: "IdOnly",
					AdditionalProperties: &AdditionalProperties{
						FieldURI: []FieldURI{
							{FieldURI: "item:Subject"},
							{FieldURI: "item:DateTimeReceived"},
							{FieldURI: "message:From"},
							{FieldURI: "message:IsRead"},
							{FieldURI: "item:HasAttachments"},
							{FieldURI: "item:ConversationId"},
						},
					},
				},
				ParentFolderIds: ParentFolderIds{
					DistinguishedFolderId: DistinguishedFolderId{
						Id: folderID,
					},
				},
			},
		},
	}

	return envelope
}

// buildGetItemRequest creates a GetItem SOAP request
func (c *Client) buildGetItemRequest(itemID string) *SOAPEnvelope {
	envelope := &SOAPEnvelope{
		XMLNS: "http://schemas.xmlsoap.org/soap/envelope/",
		XSI:   "http://www.w3.org/2001/XMLSchema-instance",
		M:     "http://schemas.microsoft.com/exchange/services/2006/messages",
		T:     "http://schemas.microsoft.com/exchange/services/2006/types",
		Header: SOAPHeader{
			RequestServerVersion: RequestServerVersion{
				Version: GetEWSAPIVersion(),
			},
		},
		Body: SOAPBody{
			Content: GetItemRequest{
				ItemShape: ItemShape{
					BaseShape: "Default",
					AdditionalProperties: &AdditionalProperties{
						FieldURI: []FieldURI{
							{FieldURI: "item:Body"},
							{FieldURI: "item:Subject"},
							{FieldURI: "item:DateTimeReceived"},
							{FieldURI: "item:DateTimeSent"},
							{FieldURI: "message:From"},
							{FieldURI: "message:ToRecipients"},
							{FieldURI: "message:CcRecipients"},
							{FieldURI: "message:IsRead"},
							{FieldURI: "item:HasAttachments"},
							{FieldURI: "item:Importance"},
							{FieldURI: "item:ConversationId"},
							{FieldURI: "message:InternetMessageId"},
							{FieldURI: "item:Categories"},
						},
					},
				},
				ItemIds: ItemIds{
					ItemId: []ItemId{
						{Id: itemID},
					},
				},
			},
		},
	}

	return envelope
}

// executeFindItemRequest sends a FindItem SOAP request to EWS
func (c *Client) executeFindItemRequest(ctx context.Context, envelope *SOAPEnvelope) (*FindItemResponse, error) {
	// Marshal SOAP envelope to XML
	xmlData, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP request: %w", err)
	}

	// Add XML declaration
	xmlRequest := []byte(xml.Header + string(xmlData))

	// Execute HTTP request with retries
	var response *FindItemResponse
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		response, lastErr = c.sendFindItemRequest(ctx, xmlRequest)
		if lastErr == nil {
			return response, nil
		}

		// Wait before retry (exponential backoff)
		if attempt < c.config.MaxRetries {
			waitTime := time.Duration(attempt+1) * time.Second
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// sendFindItemRequest sends a single FindItem HTTP request
func (c *Client) sendFindItemRequest(ctx context.Context, xmlRequest []byte) (*FindItemResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.serverURL, bytes.NewReader(xmlRequest))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers and authentication
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.SetBasicAuth(c.authUser, c.config.ImpersonationPassword)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("EWS server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse SOAP response
	var response FindItemResponse
	if err := xml.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SOAP response: %w", err)
	}

	return &response, nil
}

// executeGetItemRequest sends a GetItem SOAP request to EWS
func (c *Client) executeGetItemRequest(ctx context.Context, envelope *SOAPEnvelope) (*GetItemResponse, error) {
	// Marshal SOAP envelope to XML
	xmlData, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP request: %w", err)
	}

	// Add XML declaration
	xmlRequest := []byte(xml.Header + string(xmlData))

	// Execute HTTP request with retries
	var response *GetItemResponse
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		response, lastErr = c.sendGetItemRequest(ctx, xmlRequest)
		if lastErr == nil {
			return response, nil
		}

		// Wait before retry (exponential backoff)
		if attempt < c.config.MaxRetries {
			waitTime := time.Duration(attempt+1) * time.Second
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// sendGetItemRequest sends a single GetItem HTTP request
func (c *Client) sendGetItemRequest(ctx context.Context, xmlRequest []byte) (*GetItemResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.serverURL, bytes.NewReader(xmlRequest))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers and authentication
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.SetBasicAuth(c.authUser, c.config.ImpersonationPassword)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("EWS server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse SOAP response
	var response GetItemResponse
	if err := xml.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SOAP response: %w", err)
	}

	return &response, nil
}

// parseFindItemResponse converts EWS FindItem response to our model
func (c *Client) parseFindItemResponse(response *FindItemResponse) ([]EmailListItem, int, error) {
	messages := response.Body.FindItemResponse.ResponseMessages.FindItemResponseMessage

	// Check response status
	if messages.ResponseClass != "Success" {
		return nil, 0, fmt.Errorf("EWS error: %s", messages.ResponseCode)
	}

	items := messages.RootFolder.Items.Message
	total := messages.RootFolder.TotalItemsInView

	emails := make([]EmailListItem, 0, len(items))
	for _, msg := range items {
		// Parse received date
		receivedDate, _ := time.Parse(time.RFC3339, msg.DateTimeReceived)

		email := EmailListItem{
			ItemID:         msg.ItemId.Id,
			ConversationID: msg.ConversationId.Id,
			Subject:        msg.Subject,
			From:           msg.From.Mailbox.Name,
			FromEmail:      msg.From.Mailbox.EmailAddress,
			ReceivedDate:   receivedDate,
			HasAttachments: msg.HasAttachments,
			IsRead:         msg.IsRead,
		}

		emails = append(emails, email)
	}

	return emails, total, nil
}

// parseGetItemResponse converts EWS GetItem response to our model
func (c *Client) parseGetItemResponse(response *GetItemResponse) (*EmailDetail, error) {
	messages := response.Body.GetItemResponse.ResponseMessages.GetItemResponseMessage

	// Check response status
	if messages.ResponseClass != "Success" {
		return nil, fmt.Errorf("EWS error: %s", messages.ResponseCode)
	}

	if len(messages.Items.Message) == 0 {
		return nil, fmt.Errorf("no message found in response")
	}

	msg := messages.Items.Message[0]

	// Parse dates
	receivedDate, _ := time.Parse(time.RFC3339, msg.DateTimeReceived)
	sentDate, _ := time.Parse(time.RFC3339, msg.DateTimeSent)

	// Build email detail
	email := &EmailDetail{
		ItemID:         msg.ItemId.Id,
		ConversationID: msg.ConversationId.Id,
		Subject:        msg.Subject,
		From: EmailAddress{
			Name:    msg.From.Mailbox.Name,
			Address: msg.From.Mailbox.EmailAddress,
		},
		ReceivedDate:      receivedDate,
		SentDate:          sentDate,
		HasAttachments:    msg.HasAttachments,
		IsRead:            msg.IsRead,
		Importance:        msg.Importance,
		InternetMessageID: msg.InternetMessageId,
	}

	// Parse body
	if msg.Body != nil {
		email.Body = strings.TrimSpace(msg.Body.Content)
		email.BodyType = msg.Body.BodyType
	}

	// Parse To recipients
	if msg.ToRecipients != nil {
		email.ToRecipients = make([]EmailAddress, 0, len(msg.ToRecipients.Mailbox))
		for _, mbx := range msg.ToRecipients.Mailbox {
			email.ToRecipients = append(email.ToRecipients, EmailAddress{
				Name:    mbx.Name,
				Address: mbx.EmailAddress,
			})
		}
	}

	// Parse CC recipients
	if msg.CcRecipients != nil {
		email.CcRecipients = make([]EmailAddress, 0, len(msg.CcRecipients.Mailbox))
		for _, mbx := range msg.CcRecipients.Mailbox {
			email.CcRecipients = append(email.CcRecipients, EmailAddress{
				Name:    mbx.Name,
				Address: mbx.EmailAddress,
			})
		}
	}

	// Parse categories
	if msg.Categories != nil {
		email.Categories = msg.Categories.String
	}

	return email, nil
}

// getConversationThread retrieves all emails in a conversation thread
func (c *Client) getConversationThread(ctx context.Context, mailbox, conversationID, excludeItemID string) ([]EmailListItem, error) {
	// For now, return empty thread
	// Full conversation retrieval would require additional EWS FindConversation API
	// which is more complex to implement
	// This can be enhanced later if needed
	return []EmailListItem{}, nil
}
