package domain

import (
	"time"

	"github.com/google/uuid"
)

type Machine struct {
	ID        uuid.UUID
	Host      string
	Username  string
	CreatedAt time.Time
}
