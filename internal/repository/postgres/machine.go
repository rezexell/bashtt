package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rezexell/bashtt/internal/domain"
)

type MachineRepository struct {
	pool *pgxpool.Pool
}

func NewMachineRepository(
	db *DB,
) *MachineRepository {
	return &MachineRepository{
		pool: db.pool,
	}
}

func (r *MachineRepository) CreateMachine(
	ctx context.Context,
	machine domain.Machine,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO machines (
			id,
			host,
			username,
			created_at
		)
		VALUES ($1, $2, $3, $4)
		`,
		machine.ID,
		machine.Host,
		machine.Username,
		machine.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf(
			"insert machine: %w",
			err,
		)
	}

	return nil
}
