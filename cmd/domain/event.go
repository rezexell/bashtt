package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventAction string

const (
	EventActionOpen    EventAction = "open"
	EventActionExecute EventAction = "execute"
)

func (a EventAction) IsValid() bool {
	switch a {
	case EventActionOpen, EventActionExecute:
		return true
	default:
		return false
	}
}

type Event struct {
	ID         int64
	ScriptID   uuid.UUID
	Username   string
	ScriptPath string
	Action     EventAction
	Time       time.Time
	CreatedAt  time.Time
}
