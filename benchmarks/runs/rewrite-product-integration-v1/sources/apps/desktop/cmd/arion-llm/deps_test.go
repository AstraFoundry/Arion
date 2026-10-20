package main

import (
	"os/exec"
	"strings"
	"testing"
)

// The worker must load only its own ONNX Runtime: importing the speech
// runtime (or anything needing cgo) would put both builds in one process.
func TestWorkerDependencies(t *testing.T) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}} {{.CgoFiles}}", ".")
	cmd.Env = append(cmd.Environ(), "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.Contains(line, "sherpa") || strings.Contains(line, "mygo") || strings.Contains(line, "malgo") {
			t.Fatalf("worker depends on %s", line)
		}
	}
}
