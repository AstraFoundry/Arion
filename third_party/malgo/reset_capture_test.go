package malgo

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Exercise the actual C buffer and resampler reset without accessing hardware.
func TestResetCaptureIsolation(t *testing.T) {
	compiler := strings.Fields(os.Getenv("CC"))
	if len(compiler) == 0 {
		compiler = []string{"cc"}
		if runtime.GOOS == "windows" {
			compiler[0] = "gcc"
		}
	}
	output := filepath.Join(t.TempDir(), "reset-capture")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	args := append(compiler[1:], "-std=c99", "-DMA_ENABLE_ONLY_SPECIFIC_BACKENDS", "-DMA_ENABLE_NULL", "testdata/reset_capture.c", "-o", output, "-lm")
	if runtime.GOOS != "windows" {
		args = append(args, "-lpthread")
	}
	if runtime.GOOS == "linux" {
		args = append(args, "-ldl")
	}
	if out, err := exec.Command(compiler[0], args...).CombinedOutput(); err != nil {
		t.Fatalf("compile reset harness: %v\n%s", err, out)
	}
	if out, err := exec.Command(output).CombinedOutput(); err != nil {
		t.Fatalf("reset isolation: %v\n%s", err, out)
	}
}
