//go:build darwin

package macos

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// CLP-A07: optional geometry must not replace destination/security validation.
func TestClipboardNativeTargetIdentityWithoutGeometry(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "history-target")
	command := exec.Command("xcrun", "--sdk", "macosx", "clang", "-fobjc-arc", "testdata/history_target.m", "-o", binary,
		"-framework", "Cocoa", "-framework", "ApplicationServices", "-framework", "Carbon")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("compile history target checks: %v\n%s", err, output)
	}
	if output, err := exec.Command(binary).CombinedOutput(); err != nil {
		t.Fatalf("history target checks: %v\n%s", err, output)
	}
}

func TestClipboardUnavailableCaptureCannotDeliver(t *testing.T) {
	for _, target := range []*HistoryTarget{nil, {}, {secure: true}, {windowFallback: true}} {
		if target.Check() == nil || target.Paste(0) == nil {
			t.Fatal("an unavailable capture reached clipboard delivery")
		}
	}
}
