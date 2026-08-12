package service

import (
	"context"
	"os"

	"github.com/google/uuid"
	"github.com/rezexell/bashtt/internal/domain"
)

type SSHClient interface {
	Close() error

	Upload(
		ctx context.Context,
		path string,
		data []byte,
		mode os.FileMode,
	) error

	Execute(
		ctx context.Context,
		command string,
	) ([]byte, error)
}

type SSHFactory interface {
	Connect(
		ctx context.Context,
		host string,
		user string,
		password string,
	) (SSHClient, error)
}

type TemplateProvider interface {
	Get(ctx context.Context, template domain.Template) ([]byte, error)
}

type MachineRepository interface {
	Create(ctx context.Context, machine domain.Machine) error
}

type ScriptRepository interface {
	Create(ctx context.Context, script domain.Script) error
}

type AgentInstaller interface {
	Install(
		ctx context.Context,
		client SSHClient,
	) error
}

type CreateRequest struct {
	Host     string
	User     string
	Password string
	Template domain.Template
}

type CreateResponse struct {
	ID        uuid.UUID
	MachineID uuid.UUID
	Path      string
	Template  domain.Template
}
