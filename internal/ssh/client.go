package ssh

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

type Config struct {
	Port           int
	ConnectTimeout time.Duration
}

type Factory struct {
	config Config
}

func NewFactory(config Config) *Factory {
	return &Factory{
		config: config,
	}
}

type Client struct {
	sshClient  *gossh.Client
	sftpClient *sftp.Client
}

func (f *Factory) Connect(
	ctx context.Context,
	host string,
	user string,
	password string,
) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	address := net.JoinHostPort(
		host,
		fmt.Sprintf("%d", f.config.Port),
	)

	sshConfig := &gossh.ClientConfig{
		User: user,
		Auth: []gossh.AuthMethod{
			gossh.Password(password),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         f.config.ConnectTimeout,
	}

	dialer := &net.Dialer{
		Timeout: f.config.ConnectTimeout,
	}

	rawConn, err := dialer.DialContext(
		ctx,
		"tcp",
		address,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"dial SSH server %s: %w",
			address,
			err,
		)
	}

	clientConn, chans, reqs, err := gossh.NewClientConn(
		rawConn,
		address,
		sshConfig,
	)
	if err != nil {
		_ = rawConn.Close()

		return nil, fmt.Errorf(
			"create SSH connection to %s: %w",
			address,
			err,
		)
	}

	sshClient := gossh.NewClient(
		clientConn,
		chans,
		reqs,
	)

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()

		return nil, fmt.Errorf(
			"create SFTP client: %w",
			err,
		)
	}

	return &Client{
		sshClient:  sshClient,
		sftpClient: sftpClient,
	}, nil
}

func (c *Client) Close() error {
	var firstErr error

	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			firstErr = err
		}
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (c *Client) Upload(
	ctx context.Context,
	path string,
	data []byte,
	mode os.FileMode,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	file, err := c.sftpClient.Create(path)
	if err != nil {
		return fmt.Errorf(
			"create remote file %q: %w",
			path,
			err,
		)
	}

	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf(
			"write remote file %q: %w",
			path,
			err,
		)
	}

	if err := file.Chmod(mode); err != nil {
		return fmt.Errorf(
			"chmod remote file %q: %w",
			path,
			err,
		)
	}

	return nil
}

func (c *Client) Execute(
	ctx context.Context,
	command string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf(
			"create SSH session: %w",
			err,
		)
	}

	defer session.Close()

	type result struct {
		output []byte
		err    error
	}

	resultCh := make(chan result, 1)

	go func() {
		output, err := session.CombinedOutput(command)

		resultCh <- result{
			output: output,
			err:    err,
		}
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()

		return nil, ctx.Err()

	case result := <-resultCh:
		if result.err != nil {
			return result.output, fmt.Errorf(
				"execute remote command %q: %w",
				command,
				result.err,
			)
		}

		return result.output, nil
	}
}
