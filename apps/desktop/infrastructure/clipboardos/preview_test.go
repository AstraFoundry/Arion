package clipboardos

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"arion.local/core/domain"
)

func TestClipboardPreviewReadBoundsAndAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	content := []byte("owned synthetic file bytes")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
	actual, err := ReadPreviewFile(p, int64(len(content)))
	if err != nil || !bytes.Equal(actual, content) {
		t.Fatalf("scoped read failed: %v", err)
	}
	if _, err := ReadPreviewFile(p, int64(len(content)-1)); !errors.Is(err, domain.ErrClipOversize) {
		t.Fatal("byte limit was ignored", err)
	}
	p.FileAccess = [][]byte{[]byte("invalid credential")}
	if data, err := ReadPreviewFile(p, 1024); err == nil || len(data) != 0 {
		t.Fatal("invalid grant retried an unscoped read")
	}
	for _, target := range []string{dir, filepath.Join(dir, "link.png")} {
		if target != dir {
			if runtime.GOOS == "windows" {
				continue
			}
			if err := os.Symlink(path, target); err != nil {
				t.Fatal(err)
			}
		}
		p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{target}}
		if _, err := ReadPreviewFile(p, 1024); err == nil {
			t.Fatal("nonregular resource read")
		}
	}
	actual, err = os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, content) {
		t.Fatal("preview modified source file")
	}
}
