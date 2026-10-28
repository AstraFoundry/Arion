//go:build !windows || !amd64

package gguf

import "runtime"

func runtimeExecutable() string {
	if runtime.GOOS == "windows" {
		return "llama-server.exe"
	}
	return "llama-server"
}
