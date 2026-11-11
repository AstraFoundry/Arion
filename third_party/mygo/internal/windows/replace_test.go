//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// ReplaceFileW may stop halfway without a backup file, having removed the
// target or moved it away: the replacement, which holds the new contents,
// must then take the target's place rather than be lost.
func TestReplaceFileFinishesWhatReplaceFileWStopped(t *testing.T) {
	for _, errno := range []syscall.Errno{errorUnableToMoveReplacement, errorUnableToMoveReplacement2} {
		dir := t.TempDir()
		target, replacement := filepath.Join(dir, "a.txt"), filepath.Join(dir, ".a.txt.tmp")
		if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(replacement, []byte("new"), 0o644); err != nil {
			t.Fatal(err)
		}
		saved := replaceFileW
		replaceFileW = func(*uint16, *uint16) error {
			// What the documentation says ReplaceFileW leaves for each error.
			if errno == errorUnableToMoveReplacement {
				return firstErr(os.Remove(target), errno)
			}
			return firstErr(os.Rename(target, filepath.Join(dir, "a.txt.old")), errno)
		}
		err := new(Backend).ReplaceFile(target, replacement)
		replaceFileW = saved
		if err != nil {
			t.Fatalf("%v: %v", errno, err)
		}
		if b, err := os.ReadFile(target); err != nil || string(b) != "new" {
			t.Errorf("%v: target %q, %v", errno, b, err)
		}
		if _, err := os.Lstat(replacement); !os.IsNotExist(err) {
			t.Errorf("%v: the replacement is still there (%v)", errno, err)
		}
	}
}

func firstErr(err, otherwise error) error {
	if err != nil {
		return err
	}
	return otherwise
}
