package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rezexell/bashtt/internal/ssh"
)

func TestAgentInstaller_Install(t *testing.T) {
	if os.Getenv("BASH_TEST") != "1" {
		t.Skip(
			"set BASH_TEST=1 to run SSH integration test",
		)
	}

	port, err := strconv.Atoi(
		os.Getenv("SSH_PORT"),
	)
	if err != nil {
		t.Fatalf(
			"parse SSH_PORT: %v",
			err,
		)
	}

	host := os.Getenv("TEST_SSH_HOST")
	user := os.Getenv("TEST_SSH_USER")
	password := os.Getenv("TEST_SSH_PASSWORD")
	callbackURL := os.Getenv("AGENT_CALLBACK_URL")

	if host == "" {
		t.Fatal("TEST_SSH_HOST is empty")
	}

	if user == "" {
		t.Fatal("TEST_SSH_USER is empty")
	}

	if password == "" {
		t.Fatal("TEST_SSH_PASSWORD is empty")
	}

	if callbackURL == "" {
		t.Fatal("AGENT_CALLBACK_URL is empty")
	}

	agentPath, err := projectRootAgentPath()
	if err != nil {
		t.Fatalf(
			"resolve agent binary: %v",
			err,
		)
	}

	t.Logf(
		"agent binary: %s",
		agentPath,
	)

	if _, err := os.Stat(agentPath); err != nil {
		t.Fatalf(
			"agent binary does not exist: %v",
			err,
		)
	}

	sshFactory := ssh.NewFactory(
		ssh.Config{
			Port: port,

			ConnectTimeout: 10 * time.Second,
		},
	)

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	client, err := sshFactory.Connect(
		ctx,
		host,
		user,
		password,
	)
	if err != nil {
		t.Fatalf(
			"connect SSH: %v",
			err,
		)
	}

	defer client.Close()

	installer := NewAgentInstaller(
		AgentInstallerConfig{
			LocalBinaryPath: agentPath,

			RemoteBinaryPath: "/tmp/bashtt/agent",

			WatchDir: "/tmp/bashtt",

			CallbackURL: callbackURL,
		},
	)

	if err := installer.Install(
		ctx,
		client,
	); err != nil {
		t.Fatalf(
			"install agent: %v",
			err,
		)
	}

	output, err := client.Execute(
		ctx,
		"ps aux | grep '[b]ashtt/agent'",
	)
	if err != nil {
		t.Fatalf(
			"check agent process: %v\noutput: %s",
			err,
			output,
		)
	}

	t.Logf(
		"agent process:\n%s",
		output,
	)
}

func projectRootAgentPath() (string, error) {
	cmd := exec.Command(
		"git",
		"rev-parse",
		"--show-toplevel",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	root := strings.TrimSpace(string(output))

	return filepath.Join(
		root,
		"bin",
		"agent",
	), nil
}
