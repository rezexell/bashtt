package service

import (
	"context"

	"github.com/rezexell/bashtt/internal/ssh"
)

type SSHFactoryAdapter struct {
	factory *ssh.Factory
}

func NewSSHFactoryAdapter(
	factory *ssh.Factory,
) *SSHFactoryAdapter {
	return &SSHFactoryAdapter{
		factory: factory,
	}
}

func (a *SSHFactoryAdapter) Connect(
	ctx context.Context,
	host string,
	user string,
	password string,
) (SSHClient, error) {
	return a.factory.Connect(
		ctx,
		host,
		user,
		password,
	)
}
