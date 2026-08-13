package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/rezexell/bashtt/internal/domain"
)

const (
	defaultScriptDir  = "/tmp/bashtt"
	defaultScriptMode = 0755
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
	Get(
		ctx context.Context,
		template domain.Template,
	) ([]byte, error)
}

type MachineRepository interface {
	CreateMachine(
		ctx context.Context,
		machine domain.Machine,
	) error
}

type ScriptRepository interface {
	CreateScript(
		ctx context.Context,
		script domain.Script,
	) error
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

type CreateService struct {
	sshFactory       SSHFactory
	templateProvider TemplateProvider
	machines         MachineRepository
	scripts          ScriptRepository
	agentInstaller   AgentInstaller
}

func NewCreateService(
	sshFactory SSHFactory,
	templateProvider TemplateProvider,
	machines MachineRepository,
	scripts ScriptRepository,
	agentInstaller AgentInstaller,
) *CreateService {
	return &CreateService{
		sshFactory:       sshFactory,
		templateProvider: templateProvider,
		machines:         machines,
		scripts:          scripts,
		agentInstaller:   agentInstaller,
	}
}

func (s *CreateService) Create(
	ctx context.Context,
	req CreateRequest,
) (CreateResponse, error) {
	if err := validateCreateRequest(req); err != nil {
		return CreateResponse{}, err
	}

	templateData, err := s.templateProvider.Get(
		ctx,
		req.Template,
	)
	if err != nil {
		return CreateResponse{}, fmt.Errorf(
			"get template: %w",
			err,
		)
	}

	client, err := s.sshFactory.Connect(
		ctx,
		req.Host,
		req.User,
		req.Password,
	)
	if err != nil {
		return CreateResponse{}, fmt.Errorf(
			"connect to host %q: %w",
			req.Host,
			err,
		)
	}

	defer func() {
		_ = client.Close()
	}()

	now := time.Now().UTC()

	machine := domain.Machine{
		ID:        uuid.New(),
		Host:      req.Host,
		Username:  req.User,
		CreatedAt: now,
	}

	scriptID := uuid.New()

	script := domain.Script{
		ID:        scriptID,
		MachineID: machine.ID,
		Path: filepath.Join(
			defaultScriptDir,
			fmt.Sprintf("%s.sh", scriptID),
		),
		Template:  req.Template,
		CreatedAt: now,
	}

	// Загружаем скрипт на удалённый хост.
	if err := client.Upload(
		ctx,
		script.Path,
		templateData,
		defaultScriptMode,
	); err != nil {
		return CreateResponse{}, fmt.Errorf(
			"upload script: %w",
			err,
		)
	}

	// Пока агент реализован как no-op.
	if s.agentInstaller != nil {
		if err := s.agentInstaller.Install(
			ctx,
			client,
		); err != nil {
			return CreateResponse{}, fmt.Errorf(
				"install agent: %w",
				err,
			)
		}
	}

	if s.machines != nil {
		if err := s.machines.CreateMachine(
			ctx,
			machine,
		); err != nil {
			return CreateResponse{}, fmt.Errorf(
				"save machine: %w",
				err,
			)
		}
	}

	if s.scripts != nil {
		if err := s.scripts.CreateScript(
			ctx,
			script,
		); err != nil {
			return CreateResponse{}, fmt.Errorf(
				"save script: %w",
				err,
			)
		}
	}

	return CreateResponse{
		ID:        script.ID,
		MachineID: machine.ID,
		Path:      script.Path,
		Template:  script.Template,
	}, nil
}

func validateCreateRequest(req CreateRequest) error {
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}

	if req.User == "" {
		return fmt.Errorf("user is required")
	}

	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	if !req.Template.IsValid() {
		return domain.ErrInvalidTemplate
	}

	return nil
}
