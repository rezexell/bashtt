package memory

import (
	"context"
	"sync"

	"github.com/rezexell/bashtt/internal/domain"
)

type MachineRepository struct {
	mu       sync.RWMutex
	machines map[string]domain.Machine
}

func NewMachineRepository() *MachineRepository {
	return &MachineRepository{
		machines: make(map[string]domain.Machine),
	}
}

func (r *MachineRepository) Create(
	ctx context.Context,
	machine domain.Machine,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.machines[machine.ID.String()] = machine

	return nil
}
