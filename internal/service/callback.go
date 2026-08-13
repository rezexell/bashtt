package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/rezexell/bashtt/internal/domain"
)

type EventRepository interface {
	Create(
		ctx context.Context,
		event domain.Event,
	) error
}

type CallbackRequest struct {
	User   string
	Script string
	Action domain.EventAction
	Time   time.Time
}

type CallbackService struct {
	logger     *slog.Logger
	eventsRepo EventRepository
}

func NewCallbackService(
	logger *slog.Logger,
	eventsRepo EventRepository,
) *CallbackService {
	return &CallbackService{
		logger:     logger,
		eventsRepo: eventsRepo,
	}
}

func (s *CallbackService) Handle(
	ctx context.Context,
	req CallbackRequest,
) error {
	if err := validateCallbackRequest(req); err != nil {
		return err
	}

	event := domain.Event{
		ID: uuid.New(),

		User:   req.User,
		Script: req.Script,
		Action: req.Action,

		CreatedAt: req.Time,
	}

	if err := s.eventsRepo.Create(
		ctx,
		event,
	); err != nil {
		return fmt.Errorf(
			"save event: %w",
			err,
		)
	}

	s.logger.Info(
		"agent event received",
		"user", event.User,
		"script", event.Script,
		"action", event.Action,
		"time", event.CreatedAt,
	)

	return nil
}

func validateCallbackRequest(
	req CallbackRequest,
) error {
	if req.User == "" {
		return fmt.Errorf("user is required")
	}

	if req.Script == "" {
		return fmt.Errorf("script is required")
	}

	if !req.Action.IsValid() {
		return fmt.Errorf(
			"invalid action %q",
			req.Action,
		)
	}

	if req.Time.IsZero() {
		return fmt.Errorf("time is required")
	}

	return nil
}
