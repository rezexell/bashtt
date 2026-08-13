package agent

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rezexell/bashtt/internal/domain"
)

type FakeWatcher struct {
	script string
}

func NewFakeWatcher(
	script string,
) *FakeWatcher {
	return &FakeWatcher{
		script: script,
	}
}

func (w *FakeWatcher) Watch(
	ctx context.Context,
) (<-chan domain.Event, error) {
	events := make(chan domain.Event)

	go func() {
		defer close(events)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				events <- domain.Event{
					ID:        uuid.New(),
					User:      "root",
					Script:    w.script,
					Action:    domain.EventExecute,
					CreatedAt: time.Now().UTC(),
				}
			}
		}
	}()

	return events, nil
}
