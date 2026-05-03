package connector

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// LocalExecutor runs commands on the local machine
type LocalExecutor struct {
	os   string
	arch string
}

func NewLocalExecutor() (*LocalExecutor, error) {
	os := "linux"
	if runtime.GOOS == "darwin" {
		os = "darwin"
	}
	
	// Detect architecture
	arch := runtime.GOARCH
	
	return &LocalExecutor{os: os, arch: arch}, nil
}

func (e *LocalExecutor) Run(ctx context.Context, cmd string) (stdout string, stderr string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	// Use shell to execute commands
	command := exec.CommandContext(ctx, "sh", "-c", cmd)
	out, err := command.CombinedOutput()
	
	output := string(out)
	if err != nil {
		return "", output, err
	}
	return output, "", nil
}

func (e *LocalExecutor) OS() string   { return e.os }
func (e *LocalExecutor) Arch() string { return e.arch }
func (e *LocalExecutor) Close() error { return nil }

// detectLocalOS tries to detect the actual OS using uname
func detectLocalOS() string {
	out, err := exec.Command("uname", "-s").Output()
	if err != nil {
		return runtime.GOOS
	}
	osName := strings.ToLower(strings.TrimSpace(string(out)))
	if osName == "linux" {
		return "linux"
	}
	if osName == "darwin" {
		return "darwin"
	}
	return runtime.GOOS
}
