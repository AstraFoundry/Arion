package mygo

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestAppFileInUse: a file of the app that another program has open, as
// antivirus software scanning it, is replaced once it lets go, and read
// while it is being replaced.
func TestAppFileInUse(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows keeps open files from being replaced")
	}
	path := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Go opens files without sharing their deletion, which replacing needs.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	time.AfterFunc(500*time.Millisecond, func() { f.Close() })
	if err := writeAppFile(path, func(w io.Writer) error { _, err := io.WriteString(w, "new"); return err }); err != nil {
		t.Fatal(err)
	}
	if b, err := readAppFile(path); err != nil || string(b) != "new" {
		t.Errorf("read %q, %v after the replace", b, err)
	}
	// Reads while it is replaced over and over.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			if err := writeAppFile(path, func(w io.Writer) error { _, err := io.WriteString(w, "x"); return err }); err != nil {
				t.Errorf("write %d: %v", i, err)
				return
			}
		}
	}()
	for {
		select {
		case <-done:
			return
		default:
		}
		if _, err := readAppFile(path); err != nil {
			t.Errorf("read while replaced: %v", err)
			<-done
			return
		}
	}
}
