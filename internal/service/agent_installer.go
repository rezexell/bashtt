package service

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type AgentInstallerConfig struct {
	LocalBinaryPath  string
	RemoteBinaryPath string
	WatchDir         string
	CallbackURL      string
}

type DefaultAgentInstaller struct {
	cfg AgentInstallerConfig
}

func NewAgentInstaller(
	cfg AgentInstallerConfig,
) *DefaultAgentInstaller {
	return &DefaultAgentInstaller{
		cfg: cfg,
	}
}

func (i *DefaultAgentInstaller) Install(
	ctx context.Context,
	client SSHClient,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := i.validate(); err != nil {
		return err
	}

	agentBinary, err := os.ReadFile(
		i.cfg.LocalBinaryPath,
	)
	if err != nil {
		return fmt.Errorf(
			"read agent binary %q: %w",
			i.cfg.LocalBinaryPath,
			err,
		)
	}

	if err := i.prepareRemoteDirectory(
		ctx,
		client,
	); err != nil {
		return err
	}

	if err := i.uploadAgent(
		ctx,
		client,
		agentBinary,
	); err != nil {
		return err
	}

	if err := i.startAgent(
		ctx,
		client,
	); err != nil {
		return err
	}

	return nil
}

func (i *DefaultAgentInstaller) prepareRemoteDirectory(
	ctx context.Context,
	client SSHClient,
) error {
	command := fmt.Sprintf(
		"mkdir -p %s",
		shellQuote(i.cfg.WatchDir),
	)

	if _, err := client.Execute(
		ctx,
		command,
	); err != nil {
		return fmt.Errorf(
			"create agent directory: %w",
			err,
		)
	}

	return nil
}

func (i *DefaultAgentInstaller) uploadAgent(
	ctx context.Context,
	client SSHClient,
	data []byte,
) error {
	if err := client.Upload(
		ctx,
		i.cfg.RemoteBinaryPath,
		data,
		0755,
	); err != nil {
		return fmt.Errorf(
			"upload agent to %q: %w",
			i.cfg.RemoteBinaryPath,
			err,
		)
	}

	return nil
}

func (i *DefaultAgentInstaller) startAgent(
	ctx context.Context,
	client SSHClient,
) error {
	command := fmt.Sprintf(
		"nohup %s --watch-dir %s --callback-url %s </dev/null >%s/agent.log 2>&1 &",
		shellQuote(i.cfg.RemoteBinaryPath),
		shellQuote(i.cfg.WatchDir),
		shellQuote(i.cfg.CallbackURL),
		shellQuote(i.cfg.WatchDir),
	)

	if _, err := client.Execute(
		ctx,
		command,
	); err != nil {
		return fmt.Errorf(
			"start agent: %w",
			err,
		)
	}

	return nil
}

func (i *DefaultAgentInstaller) validate() error {
	if i.cfg.LocalBinaryPath == "" {
		return fmt.Errorf(
			"agent local binary path is empty",
		)
	}

	if i.cfg.RemoteBinaryPath == "" {
		return fmt.Errorf(
			"agent remote binary path is empty",
		)
	}

	if i.cfg.WatchDir == "" {
		return fmt.Errorf(
			"agent watch directory is empty",
		)
	}

	if i.cfg.CallbackURL == "" {
		return fmt.Errorf(
			"agent callback URL is empty",
		)
	}

	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(
		value,
		"'",
		"'\"'\"'",
	) + "'"
}
