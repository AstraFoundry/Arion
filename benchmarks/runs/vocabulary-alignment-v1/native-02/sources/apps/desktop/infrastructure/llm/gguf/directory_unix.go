//go:build !windows

package gguf

import "os"

func privateDirectory() (string, error) { return os.MkdirTemp("", "arion-llm-") }
