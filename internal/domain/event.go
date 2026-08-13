package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventAction string

const (
	EventOpen    EventAction = "open"
	EventExecute EventAction = "execute"
)

func (a EventAction) IsValid() bool {
	switch a {
	case EventOpen, EventExecute:
		return true
	default:
		return false
	}
}

type Event struct {
	ID        int64
	MachineID *uuid.UUID
	ScriptID  *uuid.UUID

	Username string
	Script   string
	Action   EventAction

	CreatedAt time.Time
}
