package memory

import (
	"context"
	"sync"

	"github.com/rezexell/bashtt/internal/domain"
)

type ScriptRepository struct {
	mu      sync.RWMutex
	scripts map[string]domain.Script
}

func NewScriptRepository() *ScriptRepository {
	return &ScriptRepository{
		scripts: make(map[string]domain.Script),
	}
}

func (r *ScriptRepository) Create(
	ctx context.Context,
	script domain.Script,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.scripts[script.ID.String()] = script

	return nil
}
