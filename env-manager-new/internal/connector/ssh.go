package connector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/config"
)

// SSHExecutor connects to remote nodes via SSH
type SSHExecutor struct {
	node   config.NodeConfig
	client *ssh.Client
	os     string
	arch   string
}

func NewSSHExecutor(node config.NodeConfig) (*SSHExecutor, error) {
	// Expand key path if needed
	keyPath := node.KeyPath
	if strings.HasPrefix(keyPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		keyPath = filepath.Join(home, keyPath[2:])
	}

	// Build SSH client config
	sshConfig, err := buildSSHConfig(node.User, node.Auth, keyPath)
	if err != nil {
		return nil, fmt.Errorf("build ssh config: %w", err)
	}

	// Connect
	host := fmt.Sprintf("%s:%d", node.Host, node.Port)
	if node.Port == 0 {
		host = fmt.Sprintf("%s:22", node.Host)
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", host, err)
	}

	exec := &SSHExecutor{
		node:   node,
		client: client,
	}

	// Detect remote OS and arch
	exec.detectRemoteInfo()

	return exec, nil
}

func (e *SSHExecutor) Run(ctx context.Context, cmd string) (stdout string, stderr string, err error) {
	session, err := e.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// Set timeout via context
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			session.Signal(ssh.SIGTERM)
			session.Close()
		case <-done:
		}
	}()

	out, err := session.CombinedOutput(cmd)
	close(done)

	output := string(out)
	if err != nil {
		return "", output, err
	}
	return output, "", nil
}

func (e *SSHExecutor) OS() string   { return e.os }
func (e *SSHExecutor) Arch() string { return e.arch }

func (e *SSHExecutor) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

func (e *SSHExecutor) detectRemoteInfo() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Detect OS
	out, _, _ := e.Run(ctx, "uname -s")
	osName := strings.ToLower(strings.TrimSpace(out))
	if osName == "linux" || osName == "darwin" {
		e.os = osName
	} else {
		e.os = "linux" // default
	}

	// Detect arch
	out, _, _ = e.Run(ctx, "uname -m")
	arch := strings.ToLower(strings.TrimSpace(out))
	switch arch {
	case "x86_64":
		e.arch = "amd64"
	case "aarch64", "arm64":
		e.arch = "arm64"
	default:
		e.arch = arch
	}
}

func buildSSHConfig(user, authType, keyPath string) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: use known_hosts
		Timeout:         10 * time.Second,
	}

	switch authType {
	case "key":
		key, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", keyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}

	case "agent":
		// Try ssh-agent
		// TODO: implement ssh-agent support
		config.Auth = []ssh.AuthMethod{ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
			return nil, fmt.Errorf("ssh-agent not yet implemented")
		})}

	case "password":
		// Not recommended, but supported
		config.Auth = []ssh.AuthMethod{ssh.Password("")} // TODO: prompt or config
		return nil, fmt.Errorf("password auth not yet implemented")

	default:
		return nil, fmt.Errorf("unknown auth type: %s", authType)
	}

	return config, nil
}
