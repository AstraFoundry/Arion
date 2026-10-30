package clipboardos

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"arion.local/core/domain"
	"github.com/egoist/mygo/transfer"
)

func TestClipboardCaptureFormatsPreserveResourcesAndExcludeInternal(t *testing.T) {
	if _, err := captureFormat([]transfer.Format{transfer.Text, Internal}); !errors.Is(err, transfer.ErrFormat) {
		t.Fatal("internal copy eligible")
	}
	text := " 你好\n🙂\r\n"
	p, err := capturePayload(transfer.TextData(text), transfer.Text, 1024)
	if err != nil || p.Text != text {
		t.Fatalf("text changed %q %v", p.Text, err)
	}
	if _, err = capturePayload(transfer.TextData(text), transfer.Text, 1); !errors.Is(err, domain.ErrClipOversize) {
		t.Fatal("oversize not reported")
	}
	paths := []string{filepath.Join(t.TempDir(), "missing.txt"), t.TempDir()}
	data, err := transfer.FileData(paths...)
	if err != nil {
		t.Fatal(err)
	}
	f, err := captureFormat(append(data.Formats(), transfer.Text))
	if err != nil || f != transfer.FileList {
		t.Fatal("file references became text")
	}
	p, err = capturePayload(data, f, 10240)
	if err != nil || len(p.Files) != 2 || p.Files[0] != paths[0] || p.Files[1] != paths[1] {
		t.Fatal("file group changed")
	}
	var pngBytes bytes.Buffer
	if err = png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	p, err = capturePayload(transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, pngBytes.Bytes()))), transfer.PNG, 10240)
	if err != nil || !bytes.Equal(p.PNG, pngBytes.Bytes()) {
		t.Fatal("source-free image changed")
	}
	if textPayload("https://example.com/path").Kind != domain.ClipLink || textPayload("https://example.com/path\nmore").Kind != domain.ClipText {
		t.Fatal("link classification mismatch")
	}
}

func TestClipboardExcludesFoldersAndKeepsOrdinaryFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ordinary.txt")
	if err := os.WriteFile(file, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	original := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{dir, file}}
	filtered, err := FilesOnly(original)
	if err != nil || len(filtered.Files) != 1 || filtered.Files[0] != file || len(original.Files) != 2 {
		t.Fatal("folder filtering changed ordinary files or original payload")
	}
	if _, err := FilesOnly(domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{dir}}); !errors.Is(err, transfer.ErrFormat) {
		t.Fatal("folder-only copy was recorded")
	}
	if err := ValidateFiles(original); !errors.Is(err, transfer.ErrFormat) {
		t.Fatal("folder-containing legacy group can be delivered")
	}
	if err := ValidateFiles(filtered); err != nil {
		t.Fatal(err)
	}
}
