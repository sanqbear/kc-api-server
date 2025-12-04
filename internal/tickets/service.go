package tickets

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrTicketNotFound   = errors.New("ticket not found")
	ErrEntryNotFound    = errors.New("entry not found")
	ErrTagNotFound      = errors.New("tag not found")
	ErrInvalidTitle     = errors.New("title is required")
	ErrInvalidEntryType = errors.New("entry_type is required")
	ErrInvalidTagName   = errors.New("tag name is required")
	ErrReferenceNotFound = errors.New("reference not found")
)

// Service defines the interface for ticket business logic
type Service interface {
	// Ticket operations
	CreateTicket(ctx context.Context, req *CreateTicketRequest, authorUserPublicID string) (*TicketDetailResponse, error)
	GetTicketByID(ctx context.Context, publicID string) (*TicketDetailResponse, error)
	ListTickets(ctx context.Context, page, limit int) (*TicketListResponseWrapper, error)
	UpdateTicket(ctx context.Context, publicID string, req *UpdateTicketRequest) (*TicketListResponse, error)
	DeleteTicket(ctx context.Context, publicID string) error
	SearchTickets(ctx context.Context, criteria *SearchTicketRequest, page, limit int) (*TicketListResponseWrapper, error)

	// Entry operations
	CreateEntry(ctx context.Context, ticketPublicID string, req *CreateEntryRequest, authorUserPublicID string) (*EntryDetailResponse, error)
	GetEntryByID(ctx context.Context, entryID int64) (*EntryDetailResponse, error)
	UpdateEntry(ctx context.Context, entryID int64, req *UpdateEntryRequest) (*EntryListResponse, error)
	DeleteEntry(ctx context.Context, entryID int64) error

	// Tag operations
	CreateTag(ctx context.Context, req *CreateTagRequest) (*TagResponse, error)
	GetTagByID(ctx context.Context, tagID int64) (*TagResponse, error)
	ListTags(ctx context.Context, page, limit int) (*TagListResponseWrapper, error)
	UpdateTag(ctx context.Context, tagID int64, req *UpdateTagRequest) (*TagResponse, error)
	DeleteTag(ctx context.Context, tagID int64) error

	// Ticket-Tag operations
	AddTagsToTicket(ctx context.Context, ticketPublicID string, req *AddTagRequest) error
	RemoveTagFromTicket(ctx context.Context, ticketPublicID string, tagID int64) error

	// Entry-Tag operations
	AddTagsToEntry(ctx context.Context, entryID int64, req *AddTagRequest) error
	RemoveTagFromEntry(ctx context.Context, entryID int64, tagID int64) error
}

type service struct {
	repo Repository
}

// NewService creates a new ticket service with the given repository
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// -------------------- Ticket Operations --------------------

func (s *service) CreateTicket(ctx context.Context, req *CreateTicketRequest, authorUserPublicID string) (*TicketDetailResponse, error) {
	if req.Title == "" {
		return nil, ErrInvalidTitle
	}

	// Set default values
	status := TicketStatusOpen
	if req.Status != nil {
		status = *req.Status
	}
	priority := TicketPriorityMedium
	if req.Priority != nil {
		priority = *req.Priority
	}
	requestType := TicketRequestTypeGeneralInquiry
	if req.RequestType != nil {
		requestType = *req.RequestType
	}

	ticket := &Ticket{
		Title:       req.Title,
		Status:      status,
		Priority:    priority,
		RequestType: requestType,
	}

	// Handle assigned user
	if req.AssignedUserID != nil && *req.AssignedUserID != "" {
		userID, err := s.repo.GetUserInternalID(ctx, *req.AssignedUserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get assigned user: %w", err)
		}
		ticket.AssignedUserID = sql.NullInt64{Int64: userID, Valid: true}
	}

	// Handle due date
	if req.DueDate != nil {
		ticket.DueDate = sql.NullTime{Time: *req.DueDate, Valid: true}
	}

	// Create ticket
	if err := s.repo.CreateTicket(ctx, ticket); err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	// Add tags if provided
	if len(req.TagIDs) > 0 {
		if err := s.repo.AddTagsToTicket(ctx, ticket.ID, req.TagIDs, nil); err != nil {
			return nil, fmt.Errorf("failed to add tags to ticket: %w", err)
		}
	}

	// Create initial entry (required)
	entryFormat := ContentFormatNone
	if req.InitialEntry.Format != nil {
		entryFormat = *req.InitialEntry.Format
	}

	payload := req.InitialEntry.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	entry := &TicketEntry{
		TicketID:  ticket.ID,
		EntryType: req.InitialEntry.EntryType,
		Format:    entryFormat,
		Payload:   payload,
	}

	// Set author user ID if provided
	if authorUserPublicID != "" {
		authorUserID, err := s.repo.GetUserInternalID(ctx, authorUserPublicID)
		if err != nil {
			return nil, fmt.Errorf("failed to get author user: %w", err)
		}
		entry.AuthorUserID = sql.NullInt64{Int64: authorUserID, Valid: true}
	}

	if req.InitialEntry.Body != nil {
		entry.Body = sql.NullString{String: *req.InitialEntry.Body, Valid: true}
	}

	if req.InitialEntry.ParentEntryID != nil {
		entry.ParentEntryID = sql.NullInt64{Int64: *req.InitialEntry.ParentEntryID, Valid: true}
	}

	if err := s.repo.CreateEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create initial entry: %w", err)
	}

	// Add tags to entry if provided
	if len(req.InitialEntry.TagIDs) > 0 {
		if err := s.repo.AddTagsToEntry(ctx, entry.ID, req.InitialEntry.TagIDs, nil); err != nil {
			return nil, fmt.Errorf("failed to add tags to entry: %w", err)
		}
	}

	// Create references if provided
	if len(req.InitialEntry.References) > 0 {
		if err := s.repo.CreateReferences(ctx, entry.ID, req.InitialEntry.References); err != nil {
			return nil, fmt.Errorf("failed to create references: %w", err)
		}
	}

	// Return detailed response
	return s.repo.GetTicketDetailByPublicID(ctx, ticket.PublicID)
}

func (s *service) GetTicketByID(ctx context.Context, publicID string) (*TicketDetailResponse, error) {
	detail, err := s.repo.GetTicketDetailByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}
	return detail, nil
}

func (s *service) ListTickets(ctx context.Context, page, limit int) (*TicketListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	tickets, totalCount, err := s.repo.ListTickets(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	var responses []TicketListResponse
	for _, ticket := range tickets {
		responses = append(responses, ticket.ToListResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &TicketListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (s *service) UpdateTicket(ctx context.Context, publicID string, req *UpdateTicketRequest) (*TicketListResponse, error) {
	existingTicket, err := s.repo.GetTicketByPublicID(ctx, publicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	if req.Title != nil && *req.Title != "" {
		existingTicket.Title = *req.Title
	}
	if req.Status != nil {
		existingTicket.Status = *req.Status
	}
	if req.Priority != nil {
		existingTicket.Priority = *req.Priority
	}
	if req.RequestType != nil {
		existingTicket.RequestType = *req.RequestType
	}
	if req.AssignedUserID != nil {
		if *req.AssignedUserID != "" {
			userID, err := s.repo.GetUserInternalID(ctx, *req.AssignedUserID)
			if err != nil {
				return nil, fmt.Errorf("failed to get assigned user: %w", err)
			}
			existingTicket.AssignedUserID = sql.NullInt64{Int64: userID, Valid: true}
		} else {
			existingTicket.AssignedUserID = sql.NullInt64{}
		}
	}
	if req.DueDate != nil {
		existingTicket.DueDate = sql.NullTime{Time: *req.DueDate, Valid: true}
	}

	if err := s.repo.UpdateTicket(ctx, publicID, existingTicket); err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	response := existingTicket.ToListResponse()
	return &response, nil
}

func (s *service) DeleteTicket(ctx context.Context, publicID string) error {
	if err := s.repo.DeleteTicket(ctx, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTicketNotFound
		}
		return fmt.Errorf("failed to delete ticket: %w", err)
	}
	return nil
}

func (s *service) SearchTickets(ctx context.Context, criteria *SearchTicketRequest, page, limit int) (*TicketListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	tickets, totalCount, err := s.repo.SearchTickets(ctx, criteria, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search tickets: %w", err)
	}

	var responses []TicketListResponse
	for _, ticket := range tickets {
		responses = append(responses, ticket.ToListResponse())
	}

	totalPages := (totalCount + limit - 1) / limit

	return &TicketListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// -------------------- Entry Operations --------------------

func (s *service) CreateEntry(ctx context.Context, ticketPublicID string, req *CreateEntryRequest, authorUserPublicID string) (*EntryDetailResponse, error) {
	ticketID, err := s.repo.GetTicketInternalID(ctx, ticketPublicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	entryFormat := ContentFormatNone
	if req.Format != nil {
		entryFormat = *req.Format
	}

	payload := req.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	entry := &TicketEntry{
		TicketID:  ticketID,
		EntryType: req.EntryType,
		Format:    entryFormat,
		Payload:   payload,
	}

	// Set author user ID if provided
	if authorUserPublicID != "" {
		authorUserID, err := s.repo.GetUserInternalID(ctx, authorUserPublicID)
		if err != nil {
			return nil, fmt.Errorf("failed to get author user: %w", err)
		}
		entry.AuthorUserID = sql.NullInt64{Int64: authorUserID, Valid: true}
	}

	if req.Body != nil {
		entry.Body = sql.NullString{String: *req.Body, Valid: true}
	}

	if req.ParentEntryID != nil {
		entry.ParentEntryID = sql.NullInt64{Int64: *req.ParentEntryID, Valid: true}
	}

	if err := s.repo.CreateEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create entry: %w", err)
	}

	// Add tags if provided
	if len(req.TagIDs) > 0 {
		if err := s.repo.AddTagsToEntry(ctx, entry.ID, req.TagIDs, nil); err != nil {
			return nil, fmt.Errorf("failed to add tags to entry: %w", err)
		}
	}

	// Create references if provided
	if len(req.References) > 0 {
		if err := s.repo.CreateReferences(ctx, entry.ID, req.References); err != nil {
			return nil, fmt.Errorf("failed to create references: %w", err)
		}
	}

	return s.repo.GetEntryDetailByID(ctx, entry.ID)
}

func (s *service) GetEntryByID(ctx context.Context, entryID int64) (*EntryDetailResponse, error) {
	detail, err := s.repo.GetEntryDetailByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEntryNotFound
		}
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	return detail, nil
}

func (s *service) UpdateEntry(ctx context.Context, entryID int64, req *UpdateEntryRequest) (*EntryListResponse, error) {
	existingEntry, err := s.repo.GetEntryByID(ctx, entryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEntryNotFound
		}
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}

	if req.Format != nil {
		existingEntry.Format = *req.Format
	}
	if req.Body != nil {
		existingEntry.Body = sql.NullString{String: *req.Body, Valid: true}
	}
	if req.Payload != nil {
		existingEntry.Payload = req.Payload
	}

	if err := s.repo.UpdateEntry(ctx, entryID, existingEntry); err != nil {
		return nil, fmt.Errorf("failed to update entry: %w", err)
	}

	response := existingEntry.ToListResponse()
	return &response, nil
}

func (s *service) DeleteEntry(ctx context.Context, entryID int64) error {
	if err := s.repo.DeleteEntry(ctx, entryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEntryNotFound
		}
		return fmt.Errorf("failed to delete entry: %w", err)
	}
	return nil
}

// -------------------- Tag Operations --------------------

func (s *service) CreateTag(ctx context.Context, req *CreateTagRequest) (*TagResponse, error) {
	if req.Name == "" {
		return nil, ErrInvalidTagName
	}

	tag := &Tag{
		Name: req.Name,
	}

	if req.ColorCode != nil {
		tag.ColorCode = sql.NullString{String: *req.ColorCode, Valid: true}
	}

	if err := s.repo.CreateTag(ctx, tag); err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	response := tag.ToResponse(nil)
	return &response, nil
}

func (s *service) GetTagByID(ctx context.Context, tagID int64) (*TagResponse, error) {
	tag, err := s.repo.GetTagByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	response := tag.ToResponse(nil)
	return &response, nil
}

func (s *service) ListTags(ctx context.Context, page, limit int) (*TagListResponseWrapper, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	tags, totalCount, err := s.repo.ListTags(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	var responses []TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse(nil))
	}

	totalPages := (totalCount + limit - 1) / limit

	return &TagListResponseWrapper{
		Data:       responses,
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func (s *service) UpdateTag(ctx context.Context, tagID int64, req *UpdateTagRequest) (*TagResponse, error) {
	existingTag, err := s.repo.GetTagByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	if req.Name != nil && *req.Name != "" {
		existingTag.Name = *req.Name
	}
	if req.ColorCode != nil {
		existingTag.ColorCode = sql.NullString{String: *req.ColorCode, Valid: true}
	}

	if err := s.repo.UpdateTag(ctx, tagID, existingTag); err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	response := existingTag.ToResponse(nil)
	return &response, nil
}

func (s *service) DeleteTag(ctx context.Context, tagID int64) error {
	if err := s.repo.DeleteTag(ctx, tagID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTagNotFound
		}
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	return nil
}

// -------------------- Ticket-Tag Operations --------------------

func (s *service) AddTagsToTicket(ctx context.Context, ticketPublicID string, req *AddTagRequest) error {
	ticketID, err := s.repo.GetTicketInternalID(ctx, ticketPublicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTicketNotFound
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	if err := s.repo.AddTagsToTicket(ctx, ticketID, req.TagIDs, req.Category); err != nil {
		return fmt.Errorf("failed to add tags to ticket: %w", err)
	}

	return nil
}

func (s *service) RemoveTagFromTicket(ctx context.Context, ticketPublicID string, tagID int64) error {
	ticketID, err := s.repo.GetTicketInternalID(ctx, ticketPublicID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTicketNotFound
		}
		return fmt.Errorf("failed to get ticket: %w", err)
	}

	if err := s.repo.RemoveTagFromTicket(ctx, ticketID, tagID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTagNotFound
		}
		return fmt.Errorf("failed to remove tag from ticket: %w", err)
	}

	return nil
}

// -------------------- Entry-Tag Operations --------------------

func (s *service) AddTagsToEntry(ctx context.Context, entryID int64, req *AddTagRequest) error {
	if _, err := s.repo.GetEntryByID(ctx, entryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEntryNotFound
		}
		return fmt.Errorf("failed to get entry: %w", err)
	}

	if err := s.repo.AddTagsToEntry(ctx, entryID, req.TagIDs, req.Category); err != nil {
		return fmt.Errorf("failed to add tags to entry: %w", err)
	}

	return nil
}

func (s *service) RemoveTagFromEntry(ctx context.Context, entryID int64, tagID int64) error {
	if _, err := s.repo.GetEntryByID(ctx, entryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEntryNotFound
		}
		return fmt.Errorf("failed to get entry: %w", err)
	}

	if err := s.repo.RemoveTagFromEntry(ctx, entryID, tagID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTagNotFound
		}
		return fmt.Errorf("failed to remove tag from entry: %w", err)
	}

	return nil
}
