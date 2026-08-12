package domain

import (
	"time"

	"github.com/google/uuid"
)

type Template string

const (
	Template1 Template = "template1"
	Template2 Template = "template2"
)

func (t Template) IsValid() bool {
	switch t {
	case Template1, Template2:
		return true
	default:
		return false
	}
}

type Script struct {
	ID        uuid.UUID
	MachineID uuid.UUID
	Path      string
	Template  Template
	CreatedAt time.Time
}
