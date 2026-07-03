package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/aetherius/platform/services/support/internal/model"
	"github.com/aetherius/platform/services/support/internal/repository"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrInvalidInput = errors.New("invalid input")
)

type SupportService struct {
	repo *repository.SupportRepository
}

func NewSupportService(repo *repository.SupportRepository) *SupportService {
	return &SupportService{repo: repo}
}

func (s *SupportService) CreateTicket(ctx context.Context, userID uuid.UUID, subject, category string, priority model.TicketPriority, messageContent string) (*model.Ticket, error) {
	if subject == "" {
		return nil, ErrInvalidInput
	}
	if messageContent == "" {
		return nil, ErrInvalidInput
	}
	if category == "" {
		category = "general"
	}
	if priority == "" {
		priority = model.PriorityMedium
	}

	now := time.Now()
	ticket := &model.Ticket{
		ID:        uuid.New(),
		UserID:    userID,
		Subject:   subject,
		Status:    model.TicketOpen,
		Priority:  priority,
		Category:  category,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.InsertTicket(ctx, ticket); err != nil {
		return nil, err
	}

	msg := &model.TicketMessage{
		ID:        uuid.New(),
		TicketID:  ticket.ID,
		UserID:    userID,
		Content:   messageContent,
		IsStaff:   false,
		CreatedAt: now,
	}

	if err := s.repo.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}

	return ticket, nil
}

func (s *SupportService) GetTicket(ctx context.Context, ticketID, userID uuid.UUID, isStaff bool) (*model.Ticket, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if ticket.UserID != userID && !isStaff {
		return nil, ErrForbidden
	}

	return ticket, nil
}

func (s *SupportService) ListTickets(ctx context.Context, userID uuid.UUID, isStaff bool) ([]*model.Ticket, error) {
	if isStaff {
		return s.repo.ListAllTickets(ctx)
	}
	return s.repo.ListTicketsByUserID(ctx, userID)
}

func (s *SupportService) AddMessage(ctx context.Context, ticketID, userID uuid.UUID, content string, isStaff bool) (*model.TicketMessage, error) {
	if content == "" {
		return nil, ErrInvalidInput
	}

	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if ticket.UserID != userID && !isStaff {
		return nil, ErrForbidden
	}

	msg := &model.TicketMessage{
		ID:        uuid.New(),
		TicketID:  ticketID,
		UserID:    userID,
		Content:   content,
		IsStaff:   isStaff,
		CreatedAt: time.Now(),
	}

	if err := s.repo.InsertMessage(ctx, msg); err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *SupportService) GetMessages(ctx context.Context, ticketID, userID uuid.UUID) ([]*model.TicketMessage, error) {
	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if ticket.UserID != userID {
		return nil, ErrForbidden
	}

	return s.repo.GetMessagesByTicketID(ctx, ticketID)
}

func (s *SupportService) UpdateStatus(ctx context.Context, ticketID, userID uuid.UUID, status model.TicketStatus, isStaff bool) error {
	if !isStaff {
		return ErrForbidden
	}

	ticket, err := s.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return err
	}

	if ticket.UserID != userID && !isStaff {
		return ErrForbidden
	}

	return s.repo.UpdateTicketStatus(ctx, ticketID, status)
}
