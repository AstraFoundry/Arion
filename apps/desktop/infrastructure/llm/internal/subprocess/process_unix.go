//go:build !windows

// Package subprocess owns the complete lifetime of an isolated LLM worker.
package subprocess

import (
	"os/exec"
	"syscall"
)

type Process struct {
	cmd   *exec.Cmd
	group bool
}

func Start(cmd *exec.Cmd) (*Process, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Process{cmd: cmd, group: true}, nil
}

// StartChild preserves the worker's process group for native descendants.
func StartChild(cmd *exec.Cmd) (*Process, error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &Process{cmd: cmd}, nil
}

// Stop also reaps descendants that outlived their owning worker.
func (p *Process) Stop() {
	if p.group {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	} else {
		_ = p.cmd.Process.Kill()
	}
}
