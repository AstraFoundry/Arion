package client

import "os/exec"

// The selected GGUF runtime is not provisioned on Windows.
func prepareProcess(cmd *exec.Cmd) {}
func stopProcessTree(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
