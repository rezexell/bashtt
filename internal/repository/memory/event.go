package memory

import (
	"context"
	"sync"

	"github.com/rezexell/bashtt/internal/domain"
)

type EventRepository struct {
	mu     sync.RWMutex
	events []domain.Event
}

func NewEventRepository() *EventRepository {
	return &EventRepository{
		events: make([]domain.Event, 0),
	}
}

func (r *EventRepository) Create(
	ctx context.Context,
	event domain.Event,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.events = append(r.events, event)

	return nil
}
