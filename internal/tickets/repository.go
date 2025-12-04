package tickets

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Repository defines the interface for ticket data access operations
type Repository interface {
	// Ticket operations
	CreateTicket(ctx context.Context, ticket *Ticket) error
	GetTicketByPublicID(ctx context.Context, publicID string) (*Ticket, error)
	GetTicketDetailByPublicID(ctx context.Context, publicID string) (*TicketDetailResponse, error)
	ListTickets(ctx context.Context, page, limit int) ([]Ticket, int, error)
	UpdateTicket(ctx context.Context, publicID string, ticket *Ticket) error
	DeleteTicket(ctx context.Context, publicID string) error
	SearchTickets(ctx context.Context, criteria *SearchTicketRequest, page, limit int) ([]Ticket, int, error)
	GetTicketInternalID(ctx context.Context, publicID string) (int64, error)
	GetUserInternalID(ctx context.Context, publicID string) (int64, error)

	// Entry operations
	CreateEntry(ctx context.Context, entry *TicketEntry) error
	GetEntryByID(ctx context.Context, entryID int64) (*TicketEntry, error)
	GetEntryDetailByID(ctx context.Context, entryID int64) (*EntryDetailResponse, error)
	ListEntriesByTicketID(ctx context.Context, ticketID int64) ([]EntryListResponse, error)
	UpdateEntry(ctx context.Context, entryID int64, entry *TicketEntry) error
	DeleteEntry(ctx context.Context, entryID int64) error

	// Tag operations
	CreateTag(ctx context.Context, tag *Tag) error
	GetTagByID(ctx context.Context, tagID int64) (*Tag, error)
	ListTags(ctx context.Context, page, limit int) ([]Tag, int, error)
	UpdateTag(ctx context.Context, tagID int64, tag *Tag) error
	DeleteTag(ctx context.Context, tagID int64) error

	// Ticket-Tag operations
	AddTagsToTicket(ctx context.Context, ticketID int64, tagIDs []int64, category *string) error
	RemoveTagFromTicket(ctx context.Context, ticketID int64, tagID int64) error
	GetTagsByTicketID(ctx context.Context, ticketID int64) ([]TagResponse, error)

	// Entry-Tag operations
	AddTagsToEntry(ctx context.Context, entryID int64, tagIDs []int64, category *string) error
	RemoveTagFromEntry(ctx context.Context, entryID int64, tagID int64) error
	GetTagsByEntryID(ctx context.Context, entryID int64) ([]TagResponse, error)

	// Reference operations
	CreateReferences(ctx context.Context, sourceEntryID int64, refs []CreateReferenceRequest) error
	GetReferencesByEntryID(ctx context.Context, entryID int64) ([]ReferenceResponse, error)
	DeleteReference(ctx context.Context, sourceEntryID int64, targetEntryID, targetTicketID, targetUserID *int64) error
}

type repository struct {
	db *sql.DB
}

// NewRepository creates a new ticket repository
func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// -------------------- Ticket Operations --------------------

func (r *repository) CreateTicket(ctx context.Context, ticket *Ticket) error {
	query := `
		INSERT INTO ticket_systems.tickets (
			title, assigned_user_id, status, priority, request_type, due_date
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, public_id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		ticket.Title,
		ticket.AssignedUserID,
		ticket.Status,
		ticket.Priority,
		ticket.RequestType,
		ticket.DueDate,
	).Scan(&ticket.ID, &ticket.PublicID, &ticket.CreatedAt, &ticket.UpdatedAt)
}

func (r *repository) GetTicketByPublicID(ctx context.Context, publicID string) (*Ticket, error) {
	query := `
		SELECT id, public_id, title, assigned_user_id, status, priority, request_type, due_date, created_at, updated_at
		FROM ticket_systems.tickets
		WHERE public_id = $1`

	ticket := &Ticket{}
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&ticket.ID,
		&ticket.PublicID,
		&ticket.Title,
		&ticket.AssignedUserID,
		&ticket.Status,
		&ticket.Priority,
		&ticket.RequestType,
		&ticket.DueDate,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (r *repository) GetTicketDetailByPublicID(ctx context.Context, publicID string) (*TicketDetailResponse, error) {
	query := `
		SELECT
			t.id, t.public_id, t.title, t.status, t.priority, t.request_type, t.due_date, t.created_at, t.updated_at,
			u.public_id, u.name
		FROM ticket_systems.tickets t
		LEFT JOIN organizations.users u ON t.assigned_user_id = u.id
		WHERE t.public_id = $1`

	var ticketID int64
	var assignedUserPublicID sql.NullString
	var assignedUserName sql.NullString
	detail := &TicketDetailResponse{}

	err := r.db.QueryRowContext(ctx, query, publicID).Scan(
		&ticketID,
		&detail.ID,
		&detail.Title,
		&detail.Status,
		&detail.Priority,
		&detail.RequestType,
		&detail.DueDate,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&assignedUserPublicID,
		&assignedUserName,
	)
	if err != nil {
		return nil, err
	}

	if assignedUserPublicID.Valid {
		detail.AssignedUserID = &assignedUserPublicID.String
		if assignedUserName.Valid {
			detail.AssignedUserName = json.RawMessage(assignedUserName.String)
		}
	}

	// Get tags
	tags, err := r.GetTagsByTicketID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	detail.Tags = tags

	// Get entries
	entries, err := r.ListEntriesByTicketID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	detail.Entries = entries

	return detail, nil
}

func (r *repository) ListTickets(ctx context.Context, page, limit int) ([]Ticket, int, error) {
	offset := (page - 1) * limit

	var totalCount int
	countQuery := `SELECT COUNT(*) FROM ticket_systems.tickets`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, public_id, title, assigned_user_id, status, priority, request_type, due_date, created_at, updated_at
		FROM ticket_systems.tickets
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		if err := rows.Scan(
			&ticket.ID,
			&ticket.PublicID,
			&ticket.Title,
			&ticket.AssignedUserID,
			&ticket.Status,
			&ticket.Priority,
			&ticket.RequestType,
			&ticket.DueDate,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, totalCount, rows.Err()
}

func (r *repository) UpdateTicket(ctx context.Context, publicID string, ticket *Ticket) error {
	query := `
		UPDATE ticket_systems.tickets SET
			title = $1,
			assigned_user_id = $2,
			status = $3,
			priority = $4,
			request_type = $5,
			due_date = $6
		WHERE public_id = $7`

	result, err := r.db.ExecContext(ctx, query,
		ticket.Title,
		ticket.AssignedUserID,
		ticket.Status,
		ticket.Priority,
		ticket.RequestType,
		ticket.DueDate,
		publicID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) DeleteTicket(ctx context.Context, publicID string) error {
	query := `DELETE FROM ticket_systems.tickets WHERE public_id = $1`

	result, err := r.db.ExecContext(ctx, query, publicID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) SearchTickets(ctx context.Context, criteria *SearchTicketRequest, page, limit int) ([]Ticket, int, error) {
	offset := (page - 1) * limit

	var conditions []string
	var args []interface{}
	argIndex := 1

	if criteria.Query != nil && *criteria.Query != "" {
		conditions = append(conditions, fmt.Sprintf("title ILIKE $%d", argIndex))
		args = append(args, "%"+*criteria.Query+"%")
		argIndex++
	}

	if len(criteria.Status) > 0 {
		placeholders := make([]string, len(criteria.Status))
		for i, s := range criteria.Status {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, s)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(criteria.Priority) > 0 {
		placeholders := make([]string, len(criteria.Priority))
		for i, p := range criteria.Priority {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, p)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("priority IN (%s)", strings.Join(placeholders, ", ")))
	}

	if len(criteria.RequestType) > 0 {
		placeholders := make([]string, len(criteria.RequestType))
		for i, rt := range criteria.RequestType {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, rt)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("request_type IN (%s)", strings.Join(placeholders, ", ")))
	}

	if criteria.AssignedUserID != nil && *criteria.AssignedUserID != "" {
		conditions = append(conditions, fmt.Sprintf("assigned_user_id = (SELECT id FROM organizations.users WHERE public_id = $%d)", argIndex))
		args = append(args, *criteria.AssignedUserID)
		argIndex++
	}

	if criteria.DueDateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("due_date >= $%d", argIndex))
		args = append(args, *criteria.DueDateFrom)
		argIndex++
	}

	if criteria.DueDateTo != nil {
		conditions = append(conditions, fmt.Sprintf("due_date <= $%d", argIndex))
		args = append(args, *criteria.DueDateTo)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Handle tag filtering with subquery
	if len(criteria.TagIDs) > 0 {
		tagPlaceholders := make([]string, len(criteria.TagIDs))
		for i, tagID := range criteria.TagIDs {
			tagPlaceholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, tagID)
			argIndex++
		}
		tagCondition := fmt.Sprintf("id IN (SELECT ticket_id FROM ticket_systems.ticket_tags WHERE tag_id IN (%s))", strings.Join(tagPlaceholders, ", "))
		if whereClause == "" {
			whereClause = "WHERE " + tagCondition
		} else {
			whereClause += " AND " + tagCondition
		}
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ticket_systems.tickets %s", whereClause)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, public_id, title, assigned_user_id, status, priority, request_type, due_date, created_at, updated_at
		FROM ticket_systems.tickets
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		if err := rows.Scan(
			&ticket.ID,
			&ticket.PublicID,
			&ticket.Title,
			&ticket.AssignedUserID,
			&ticket.Status,
			&ticket.Priority,
			&ticket.RequestType,
			&ticket.DueDate,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, totalCount, rows.Err()
}

func (r *repository) GetTicketInternalID(ctx context.Context, publicID string) (int64, error) {
	var id int64
	query := `SELECT id FROM ticket_systems.tickets WHERE public_id = $1`
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(&id)
	return id, err
}

func (r *repository) GetUserInternalID(ctx context.Context, publicID string) (int64, error) {
	var id int64
	query := `SELECT id FROM organizations.users WHERE public_id = $1`
	err := r.db.QueryRowContext(ctx, query, publicID).Scan(&id)
	return id, err
}

// -------------------- Entry Operations --------------------

func (r *repository) CreateEntry(ctx context.Context, entry *TicketEntry) error {
	query := `
		INSERT INTO ticket_systems.ticket_entries (
			ticket_id, author_user_id, parent_entry_id, entry_type, format, body, payload
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		entry.TicketID,
		entry.AuthorUserID,
		entry.ParentEntryID,
		entry.EntryType,
		entry.Format,
		entry.Body,
		entry.Payload,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)
}

func (r *repository) GetEntryByID(ctx context.Context, entryID int64) (*TicketEntry, error) {
	query := `
		SELECT id, ticket_id, author_user_id, parent_entry_id, entry_type, format, body, payload, is_deleted, created_at, updated_at
		FROM ticket_systems.ticket_entries
		WHERE id = $1 AND is_deleted = false`

	entry := &TicketEntry{}
	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&entry.ID,
		&entry.TicketID,
		&entry.AuthorUserID,
		&entry.ParentEntryID,
		&entry.EntryType,
		&entry.Format,
		&entry.Body,
		&entry.Payload,
		&entry.IsDeleted,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *repository) GetEntryDetailByID(ctx context.Context, entryID int64) (*EntryDetailResponse, error) {
	query := `
		SELECT
			e.id, t.public_id, e.entry_type, e.format, e.body, e.payload, e.parent_entry_id, e.created_at, e.updated_at,
			u.public_id, u.name
		FROM ticket_systems.ticket_entries e
		JOIN ticket_systems.tickets t ON e.ticket_id = t.id
		LEFT JOIN organizations.users u ON e.author_user_id = u.id
		WHERE e.id = $1 AND e.is_deleted = false`

	var authorUserPublicID sql.NullString
	var authorUserName sql.NullString
	var body sql.NullString
	var parentEntryID sql.NullInt64
	detail := &EntryDetailResponse{}

	err := r.db.QueryRowContext(ctx, query, entryID).Scan(
		&detail.ID,
		&detail.TicketID,
		&detail.EntryType,
		&detail.Format,
		&body,
		&detail.Payload,
		&parentEntryID,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&authorUserPublicID,
		&authorUserName,
	)
	if err != nil {
		return nil, err
	}

	if body.Valid {
		detail.Body = &body.String
	}
	if parentEntryID.Valid {
		detail.ParentEntryID = &parentEntryID.Int64
	}
	if authorUserPublicID.Valid {
		detail.AuthorUserID = &authorUserPublicID.String
		if authorUserName.Valid {
			detail.AuthorUserName = json.RawMessage(authorUserName.String)
		}
	}

	// Get tags
	tags, err := r.GetTagsByEntryID(ctx, entryID)
	if err != nil {
		return nil, err
	}
	detail.Tags = tags

	// Get references
	refs, err := r.GetReferencesByEntryID(ctx, entryID)
	if err != nil {
		return nil, err
	}
	detail.References = refs

	return detail, nil
}

func (r *repository) ListEntriesByTicketID(ctx context.Context, ticketID int64) ([]EntryListResponse, error) {
	query := `
		SELECT
			e.id, e.entry_type, e.format, e.body, e.parent_entry_id, e.created_at, e.updated_at,
			u.public_id, u.name
		FROM ticket_systems.ticket_entries e
		LEFT JOIN organizations.users u ON e.author_user_id = u.id
		WHERE e.ticket_id = $1 AND e.is_deleted = false
		ORDER BY e.created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []EntryListResponse
	for rows.Next() {
		var entry EntryListResponse
		var body sql.NullString
		var parentEntryID sql.NullInt64
		var authorUserPublicID sql.NullString
		var authorUserName sql.NullString

		if err := rows.Scan(
			&entry.ID,
			&entry.EntryType,
			&entry.Format,
			&body,
			&parentEntryID,
			&entry.CreatedAt,
			&entry.UpdatedAt,
			&authorUserPublicID,
			&authorUserName,
		); err != nil {
			return nil, err
		}

		if body.Valid {
			entry.Body = &body.String
		}
		if parentEntryID.Valid {
			entry.ParentEntryID = &parentEntryID.Int64
		}
		if authorUserPublicID.Valid {
			entry.AuthorUserID = &authorUserPublicID.String
			if authorUserName.Valid {
				entry.AuthorUserName = json.RawMessage(authorUserName.String)
			}
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func (r *repository) UpdateEntry(ctx context.Context, entryID int64, entry *TicketEntry) error {
	query := `
		UPDATE ticket_systems.ticket_entries SET
			format = $1,
			body = $2,
			payload = $3
		WHERE id = $4 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query,
		entry.Format,
		entry.Body,
		entry.Payload,
		entryID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) DeleteEntry(ctx context.Context, entryID int64) error {
	query := `UPDATE ticket_systems.ticket_entries SET is_deleted = true WHERE id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, entryID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// -------------------- Tag Operations --------------------

func (r *repository) CreateTag(ctx context.Context, tag *Tag) error {
	query := `
		INSERT INTO ticket_systems.tags (name, color_code)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		tag.Name,
		tag.ColorCode,
	).Scan(&tag.ID, &tag.CreatedAt, &tag.UpdatedAt)
}

func (r *repository) GetTagByID(ctx context.Context, tagID int64) (*Tag, error) {
	query := `
		SELECT id, name, color_code, is_deleted, created_at, updated_at
		FROM ticket_systems.tags
		WHERE id = $1 AND is_deleted = false`

	tag := &Tag{}
	err := r.db.QueryRowContext(ctx, query, tagID).Scan(
		&tag.ID,
		&tag.Name,
		&tag.ColorCode,
		&tag.IsDeleted,
		&tag.CreatedAt,
		&tag.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tag, nil
}

func (r *repository) ListTags(ctx context.Context, page, limit int) ([]Tag, int, error) {
	offset := (page - 1) * limit

	var totalCount int
	countQuery := `SELECT COUNT(*) FROM ticket_systems.tags WHERE is_deleted = false`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, color_code, is_deleted, created_at, updated_at
		FROM ticket_systems.tags
		WHERE is_deleted = false
		ORDER BY name ASC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.ColorCode,
			&tag.IsDeleted,
			&tag.CreatedAt,
			&tag.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tags = append(tags, tag)
	}

	return tags, totalCount, rows.Err()
}

func (r *repository) UpdateTag(ctx context.Context, tagID int64, tag *Tag) error {
	query := `
		UPDATE ticket_systems.tags SET
			name = $1,
			color_code = $2
		WHERE id = $3 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query,
		tag.Name,
		tag.ColorCode,
		tagID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) DeleteTag(ctx context.Context, tagID int64) error {
	query := `UPDATE ticket_systems.tags SET is_deleted = true WHERE id = $1 AND is_deleted = false`

	result, err := r.db.ExecContext(ctx, query, tagID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// -------------------- Ticket-Tag Operations --------------------

func (r *repository) AddTagsToTicket(ctx context.Context, ticketID int64, tagIDs []int64, category *string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	query := `INSERT INTO ticket_systems.ticket_tags (ticket_id, tag_id, category) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`

	for _, tagID := range tagIDs {
		_, err := r.db.ExecContext(ctx, query, ticketID, tagID, category)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) RemoveTagFromTicket(ctx context.Context, ticketID int64, tagID int64) error {
	query := `DELETE FROM ticket_systems.ticket_tags WHERE ticket_id = $1 AND tag_id = $2`

	result, err := r.db.ExecContext(ctx, query, ticketID, tagID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) GetTagsByTicketID(ctx context.Context, ticketID int64) ([]TagResponse, error) {
	query := `
		SELECT t.id, t.name, t.color_code, tt.category
		FROM ticket_systems.tags t
		JOIN ticket_systems.ticket_tags tt ON t.id = tt.tag_id
		WHERE tt.ticket_id = $1 AND t.is_deleted = false`

	rows, err := r.db.QueryContext(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagResponse
	for rows.Next() {
		var tag TagResponse
		var colorCode sql.NullString
		var category sql.NullString

		if err := rows.Scan(&tag.ID, &tag.Name, &colorCode, &category); err != nil {
			return nil, err
		}

		if colorCode.Valid {
			tag.ColorCode = &colorCode.String
		}
		if category.Valid {
			tag.Category = &category.String
		}

		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// -------------------- Entry-Tag Operations --------------------

func (r *repository) AddTagsToEntry(ctx context.Context, entryID int64, tagIDs []int64, category *string) error {
	if len(tagIDs) == 0 {
		return nil
	}

	query := `INSERT INTO ticket_systems.entry_tags (entry_id, tag_id, category) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`

	for _, tagID := range tagIDs {
		_, err := r.db.ExecContext(ctx, query, entryID, tagID, category)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) RemoveTagFromEntry(ctx context.Context, entryID int64, tagID int64) error {
	query := `DELETE FROM ticket_systems.entry_tags WHERE entry_id = $1 AND tag_id = $2`

	result, err := r.db.ExecContext(ctx, query, entryID, tagID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *repository) GetTagsByEntryID(ctx context.Context, entryID int64) ([]TagResponse, error) {
	query := `
		SELECT t.id, t.name, t.color_code, et.category
		FROM ticket_systems.tags t
		JOIN ticket_systems.entry_tags et ON t.id = et.tag_id
		WHERE et.entry_id = $1 AND t.is_deleted = false`

	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []TagResponse
	for rows.Next() {
		var tag TagResponse
		var colorCode sql.NullString
		var category sql.NullString

		if err := rows.Scan(&tag.ID, &tag.Name, &colorCode, &category); err != nil {
			return nil, err
		}

		if colorCode.Valid {
			tag.ColorCode = &colorCode.String
		}
		if category.Valid {
			tag.Category = &category.String
		}

		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// -------------------- Reference Operations --------------------

func (r *repository) CreateReferences(ctx context.Context, sourceEntryID int64, refs []CreateReferenceRequest) error {
	if len(refs) == 0 {
		return nil
	}

	query := `
		INSERT INTO ticket_systems.entry_references (source_entry_id, target_entry_id, target_ticket_id, target_user_id)
		VALUES ($1, $2, $3, $4)`

	for _, ref := range refs {
		var targetTicketID, targetUserID sql.NullInt64

		if ref.TargetTicketID != nil {
			ticketInternalID, err := r.GetTicketInternalID(ctx, *ref.TargetTicketID)
			if err != nil {
				return err
			}
			targetTicketID = sql.NullInt64{Int64: ticketInternalID, Valid: true}
		}

		if ref.TargetUserID != nil {
			userInternalID, err := r.GetUserInternalID(ctx, *ref.TargetUserID)
			if err != nil {
				return err
			}
			targetUserID = sql.NullInt64{Int64: userInternalID, Valid: true}
		}

		var targetEntryID sql.NullInt64
		if ref.TargetEntryID != nil {
			targetEntryID = sql.NullInt64{Int64: *ref.TargetEntryID, Valid: true}
		}

		_, err := r.db.ExecContext(ctx, query, sourceEntryID, targetEntryID, targetTicketID, targetUserID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) GetReferencesByEntryID(ctx context.Context, entryID int64) ([]ReferenceResponse, error) {
	query := `
		SELECT
			er.target_entry_id,
			t.public_id,
			u.public_id,
			u.name,
			er.created_at
		FROM ticket_systems.entry_references er
		LEFT JOIN ticket_systems.tickets t ON er.target_ticket_id = t.id
		LEFT JOIN organizations.users u ON er.target_user_id = u.id
		WHERE er.source_entry_id = $1`

	rows, err := r.db.QueryContext(ctx, query, entryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []ReferenceResponse
	for rows.Next() {
		var ref ReferenceResponse
		var targetEntryID sql.NullInt64
		var targetTicketPublicID sql.NullString
		var targetUserPublicID sql.NullString
		var targetUserName sql.NullString

		if err := rows.Scan(
			&targetEntryID,
			&targetTicketPublicID,
			&targetUserPublicID,
			&targetUserName,
			&ref.CreatedAt,
		); err != nil {
			return nil, err
		}

		if targetEntryID.Valid {
			ref.TargetType = "entry"
			ref.TargetEntryID = &targetEntryID.Int64
		} else if targetTicketPublicID.Valid {
			ref.TargetType = "ticket"
			ref.TargetTicketID = &targetTicketPublicID.String
		} else if targetUserPublicID.Valid {
			ref.TargetType = "user"
			ref.TargetUserID = &targetUserPublicID.String
			if targetUserName.Valid {
				ref.TargetUserName = json.RawMessage(targetUserName.String)
			}
		}

		refs = append(refs, ref)
	}

	return refs, rows.Err()
}

func (r *repository) DeleteReference(ctx context.Context, sourceEntryID int64, targetEntryID, targetTicketID, targetUserID *int64) error {
	var conditions []string
	var args []interface{}
	argIndex := 1

	conditions = append(conditions, fmt.Sprintf("source_entry_id = $%d", argIndex))
	args = append(args, sourceEntryID)
	argIndex++

	if targetEntryID != nil {
		conditions = append(conditions, fmt.Sprintf("target_entry_id = $%d", argIndex))
		args = append(args, *targetEntryID)
		argIndex++
	}
	if targetTicketID != nil {
		conditions = append(conditions, fmt.Sprintf("target_ticket_id = $%d", argIndex))
		args = append(args, *targetTicketID)
		argIndex++
	}
	if targetUserID != nil {
		conditions = append(conditions, fmt.Sprintf("target_user_id = $%d", argIndex))
		args = append(args, *targetUserID)
	}

	query := fmt.Sprintf("DELETE FROM ticket_systems.entry_references WHERE %s", strings.Join(conditions, " AND "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
