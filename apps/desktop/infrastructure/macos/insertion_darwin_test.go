//go:build darwin

package macos

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNativePasteboardTransaction(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "pasteboard-transaction")
	command := exec.Command("xcrun", "--sdk", "macosx", "clang", "-fobjc-arc", "testdata/pasteboard.m", "-o", binary,
		"-framework", "Cocoa", "-framework", "ApplicationServices", "-framework", "Carbon")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile pasteboard checks: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("pasteboard checks: %v\n%s", err, output)
	}
}
