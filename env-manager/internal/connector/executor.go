package connector

import (
	"context"

	"github.com/zfanmy/dreammoon-skills/skills/env-manager/internal/config"
)

// Executor is the interface for executing commands on nodes
type Executor interface {
	Run(ctx context.Context, cmd string) (stdout string, stderr string, err error)
	OS() string   // linux | darwin
	Arch() string // amd64 | arm64
	Close() error
}

// NewExecutor creates an executor based on node configuration
func NewExecutor(node config.NodeConfig) (Executor, error) {
	if node.Local || node.Host == "localhost" || node.Host == "127.0.0.1" {
		return NewLocalExecutor()
	}
	return NewSSHExecutor(node)
}
