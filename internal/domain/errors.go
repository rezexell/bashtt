package domain

import "errors"

var (
	ErrInvalidTemplate = errors.New("invalid template")
	ErrInvalidAction   = errors.New("invalid event action")
)
