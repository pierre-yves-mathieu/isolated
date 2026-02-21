package lxc

import (
	"os/exec"
)

// Executor interface for running LXC commands (allows mocking)
type Executor interface {
	Run(args ...string) ([]byte, error)
	RunCombined(args ...string) ([]byte, error)
}

// RealExecutor executes actual LXC commands
type RealExecutor struct{}

func (e *RealExecutor) Run(args ...string) ([]byte, error) {
	cmd := exec.Command("lxc", args...)
	return cmd.Output()
}

func (e *RealExecutor) RunCombined(args ...string) ([]byte, error) {
	cmd := exec.Command("lxc", args...)
	return cmd.CombinedOutput()
}

// DefaultExecutor is the executor used by default
var DefaultExecutor Executor = &RealExecutor{}

// SetExecutor sets the executor (for testing)
func SetExecutor(e Executor) {
	DefaultExecutor = e
}

// ResetExecutor resets to the real executor
func ResetExecutor() {
	DefaultExecutor = &RealExecutor{}
}
