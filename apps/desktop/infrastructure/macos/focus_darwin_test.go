//go:build darwin

package macos

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativeFocusAbsence(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "focus-observation")
	command := exec.Command("xcrun", "--sdk", "macosx", "clang", "-fobjc-arc", "testdata/focus.m", "-o", binary,
		"-framework", "Foundation", "-framework", "ApplicationServices")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile focus checks: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("focus checks: %v\n%s", err, output)
	}
}
