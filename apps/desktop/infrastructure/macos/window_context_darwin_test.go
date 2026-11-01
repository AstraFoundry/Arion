//go:build darwin

package macos

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeWindowContext(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "window-context")
	command := exec.Command("xcrun", "--sdk", "macosx", "clang", "-fobjc-arc", "testdata/window_context.m", "-o", binary,
		"-framework", "Foundation", "-framework", "ApplicationServices")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile window context checks: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("window context checks: %v\n%s", err, output)
	}
}
