package agent

import (
	"context"

	"github.com/rezexell/bashtt/internal/service"
)

type NoopInstaller struct{}

func NewNoopInstaller() *NoopInstaller {
	return &NoopInstaller{}
}

func (i *NoopInstaller) Install(
	ctx context.Context,
	client service.SSHClient,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}
