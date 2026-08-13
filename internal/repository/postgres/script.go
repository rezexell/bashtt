package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rezexell/bashtt/internal/domain"
)

type ScriptRepository struct {
	pool *pgxpool.Pool
}

func NewScriptRepository(
	db *DB,
) *ScriptRepository {
	return &ScriptRepository{
		pool: db.pool,
	}
}

func (r *ScriptRepository) CreateScript(
	ctx context.Context,
	script domain.Script,
) error {
	_, err := r.pool.Exec(
		ctx,
		`
		INSERT INTO scripts (
			id,
			machine_id,
			path,
			template,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5)
		`,
		script.ID,
		script.MachineID,
		script.Path,
		script.Template,
		script.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf(
			"insert script: %w",
			err,
		)
	}

	return nil
}

func (r *ScriptRepository) GetScriptByPath(
	ctx context.Context,
	path string,
) (domain.Script, error) {
	var script domain.Script

	err := r.pool.QueryRow(
		ctx,
		`
		SELECT
			id,
			machine_id,
			path,
			template,
			created_at
		FROM scripts
		WHERE path = $1
		`,
		path,
	).Scan(
		&script.ID,
		&script.MachineID,
		&script.Path,
		&script.Template,
		&script.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Script{}, fmt.Errorf(
				"script %q not found",
				path,
			)
		}

		return domain.Script{}, fmt.Errorf(
			"get script by path: %w",
			err,
		)
	}

	return script, nil
}
