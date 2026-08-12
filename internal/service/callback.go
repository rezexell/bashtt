package service

import (
	"context"

	"github.com/rezexell/bashtt/internal/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event domain.Event) error
}

type CallbackService struct {
	events EventRepository
}

func NewCallbackService(events EventRepository) *CallbackService {
	return &CallbackService{
		events: events,
	}
}

func (s *CallbackService) HandleEvent(
	ctx context.Context,
	event domain.Event,
) error {
	if !event.Action.IsValid() {
		return domain.ErrInvalidAction
	}

	return s.events.Create(ctx, event)
}
