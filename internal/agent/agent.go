package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/rezexell/bashtt/internal/domain"
)

type EventWatcher interface {
	Watch(
		ctx context.Context,
	) (<-chan domain.Event, error)
}

type EventSender interface {
	Send(
		ctx context.Context,
		event domain.Event,
	) error
}

type Agent struct {
	logger  *slog.Logger
	watcher EventWatcher
	sender  EventSender
}

func New(
	logger *slog.Logger,
	watcher EventWatcher,
	sender EventSender,
) *Agent {
	return &Agent{
		logger:  logger,
		watcher: watcher,
		sender:  sender,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	events, err := a.watcher.Watch(ctx)
	if err != nil {
		return fmt.Errorf("start event watcher: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-events:
			if !ok {
				return nil
			}

			if err := a.sender.Send(ctx, event); err != nil {
				a.logger.Error(
					"send agent event failed",
					"error", err,
					"user", event.User,
					"script", event.Script,
					"action", event.Action,
				)

				continue
			}

			a.logger.Info(
				"agent event sent",
				"user", event.User,
				"script", event.Script,
				"action", event.Action,
			)
		}
	}
}
