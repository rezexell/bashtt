package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rezexell/bashtt/internal/domain"
)

type EventRepository struct {
	pool *pgxpool.Pool
}

func NewEventRepository(
	db *DB,
) *EventRepository {
	return &EventRepository{
		pool: db.pool,
	}
}

func (r *EventRepository) CreateEvent(
	ctx context.Context,
	event domain.Event,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO events (
			id,
			machine_id,
			script_id,
			username,
			script_path,
			action,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		event.ID,
		event.MachineID,
		event.ScriptID,
		event.User,
		event.Script,
		event.Action,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf(
			"insert event: %w",
			err,
		)
	}

	return nil
}
