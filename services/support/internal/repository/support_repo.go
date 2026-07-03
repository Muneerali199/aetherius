package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aetherius/platform/services/support/internal/model"
)

var (
	ErrTicketNotFound  = errors.New("ticket not found")
	ErrMessageNotFound = errors.New("message not found")
)

type SupportRepository struct {
	pool *pgxpool.Pool
}

func NewSupportRepository(pool *pgxpool.Pool) *SupportRepository {
	return &SupportRepository{pool: pool}
}

func (r *SupportRepository) InsertTicket(ctx context.Context, ticket *model.Ticket) error {
	query := `
		INSERT INTO support_tickets (id, user_id, subject, status, priority, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		ticket.ID, ticket.UserID, ticket.Subject, ticket.Status,
		ticket.Priority, ticket.Category, ticket.CreatedAt, ticket.UpdatedAt,
	).Scan(&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt)
}

func (r *SupportRepository) UpdateTicketStatus(ctx context.Context, ticketID uuid.UUID, status model.TicketStatus) error {
	query := `UPDATE support_tickets SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.pool.Exec(ctx, query, status, ticketID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrTicketNotFound
	}
	return nil
}

func (r *SupportRepository) GetTicketByID(ctx context.Context, ticketID uuid.UUID) (*model.Ticket, error) {
	query := `
		SELECT id, user_id, subject, status, priority, category, created_at, updated_at
		FROM support_tickets WHERE id = $1`

	ticket := &model.Ticket{}
	err := r.pool.QueryRow(ctx, query, ticketID).Scan(
		&ticket.ID, &ticket.UserID, &ticket.Subject, &ticket.Status,
		&ticket.Priority, &ticket.Category, &ticket.CreatedAt, &ticket.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTicketNotFound
	}
	return ticket, err
}

func (r *SupportRepository) ListTicketsByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Ticket, error) {
	query := `
		SELECT id, user_id, subject, status, priority, category, created_at, updated_at
		FROM support_tickets WHERE user_id = $1
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		t := &model.Ticket{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Subject, &t.Status, &t.Priority, &t.Category, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *SupportRepository) ListAllTickets(ctx context.Context) ([]*model.Ticket, error) {
	query := `
		SELECT id, user_id, subject, status, priority, category, created_at, updated_at
		FROM support_tickets
		ORDER BY updated_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []*model.Ticket
	for rows.Next() {
		t := &model.Ticket{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Subject, &t.Status, &t.Priority, &t.Category, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (r *SupportRepository) InsertMessage(ctx context.Context, msg *model.TicketMessage) error {
	query := `
		INSERT INTO support_messages (id, ticket_id, user_id, content, is_staff, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		msg.ID, msg.TicketID, msg.UserID, msg.Content, msg.IsStaff, msg.CreatedAt,
	).Scan(&msg.ID, &msg.CreatedAt)
}

func (r *SupportRepository) GetMessagesByTicketID(ctx context.Context, ticketID uuid.UUID) ([]*model.TicketMessage, error) {
	query := `
		SELECT id, ticket_id, user_id, content, is_staff, created_at
		FROM support_messages WHERE ticket_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.TicketMessage
	for rows.Next() {
		m := &model.TicketMessage{}
		if err := rows.Scan(&m.ID, &m.TicketID, &m.UserID, &m.Content, &m.IsStaff, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
