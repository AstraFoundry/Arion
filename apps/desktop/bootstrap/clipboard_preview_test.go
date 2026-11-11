package bootstrap

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
	"golang.org/x/image/bmp"
)

func writeClipImage(t *testing.T, path string, width, height int) {
	t.Helper()
	var data bytes.Buffer
	if err := png.Encode(&data, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}

// CLP-A06: image files use in-memory previews while preserving file resources.
func TestClipboardFileThumbnailFormatsAndResourceIdentity(t *testing.T) {
	pixels := image.NewRGBA(image.Rect(0, 0, 1280, 640))
	formats := map[string]func(*bytes.Buffer) error{
		"PNG":  func(b *bytes.Buffer) error { return png.Encode(b, pixels) },
		"jpg":  func(b *bytes.Buffer) error { return jpeg.Encode(b, pixels, nil) },
		"jpeg": func(b *bytes.Buffer) error { return jpeg.Encode(b, pixels, nil) },
		"gif":  func(b *bytes.Buffer) error { return gif.Encode(b, pixels, nil) },
		"bmp":  func(b *bytes.Buffer) error { return bmp.Encode(b, pixels) },
		"webp": func(b *bytes.Buffer) error {
			data, err := base64.StdEncoding.DecodeString("UklGRiQAAABXRUJQVlA4IBgAAAAwAQCdASoBAAEAAwA0JaQAA3AA/vuUAAA=")
			if err == nil {
				_, err = b.Write(data)
			}
			return err
		},
	}
	for ext, encode := range formats {
		t.Run(ext, func(t *testing.T) {
			var data bytes.Buffer
			if err := encode(&data); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "Photo."+ext)
			if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			payload := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
			item := domain.Clip{ID: payload.ID(), Payload: payload}
			meta := clipboardMetadata([]domain.Clip{item}, nil)
			thumb := meta[item.ID].Thumbnail
			if thumb == nil || !meta[item.ID].Files[0].Available {
				t.Fatal("image file has no usable thumbnail")
			}
			w, h := thumb.Size()
			if ext == "webp" {
				if w != 1 || h != 1 {
					t.Fatal("small image was upscaled")
				}
			} else if w != 96 || h != 48 {
				t.Fatalf("thumbnail = %dx%d", w, h)
			}
			preview, err := clipPreviewBitmap(payload, 640)
			if err != nil {
				t.Fatal(err)
			}
			w, h = preview.Size()
			if w > 640 || h > 640 {
				t.Fatal("hover image exceeded its bound")
			}
			if item.Payload.Kind != domain.ClipFiles || len(item.Payload.PNG) != 0 || item.ID != payload.ID() {
				t.Fatal("preview changed file delivery or history identity")
			}
		})
	}
}

func TestClipboardFileThumbnailRefreshAndUnsupportedResources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	writeClipImage(t, path, 160, 80)
	payload := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
	item := domain.Clip{ID: payload.ID(), Payload: payload}
	meta := clipboardMetadata([]domain.Clip{item}, nil)
	unchanged := clipboardMetadata([]domain.Clip{item}, meta)
	if unchanged[item.ID].Thumbnail != meta[item.ID].Thumbnail {
		t.Fatal("unchanged file thumbnail was decoded again")
	}
	writeClipImage(t, path, 80, 160)
	fresh := clipboardMetadata([]domain.Clip{item}, meta)
	if fresh[item.ID].Thumbnail == meta[item.ID].Thumbnail {
		t.Fatal("file preview reused a stale path-keyed bitmap")
	}
	if w, h := fresh[item.ID].Thumbnail.Size(); w != 48 || h != 96 {
		t.Fatalf("replacement thumbnail = %dx%d", w, h)
	}
	item.Payload.FileAccess = [][]byte{[]byte("invalid replacement grant")}
	rejected := clipboardMetadata([]domain.Clip{item}, fresh)
	if rejected[item.ID].Thumbnail != nil {
		t.Fatal("changed credentials reused a cached thumbnail")
	}
	item.Payload.FileAccess = nil
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	missing := clipboardMetadata([]domain.Clip{item}, fresh)
	if missing[item.ID].Thumbnail != nil || missing[item.ID].Files[0].Available {
		t.Fatal("missing file retained a valid preview")
	}
	for _, p := range []domain.ClipPayload{
		{Kind: domain.ClipFiles, Files: []string{path, path + ".jpg"}},
		{Kind: domain.ClipFiles, Files: []string{path + ".pdf"}},
	} {
		record := domain.Clip{ID: p.ID(), Payload: p}
		m := clipboardMetadata([]domain.Clip{record}, nil)
		if m[record.ID].Thumbnail != nil {
			t.Fatal("file group or ordinary file requested an image decode")
		}
	}
}

func TestClipboardFilePreviewRejectsCorruptionAndOversize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "photo.png")
	p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
	if err := os.WriteFile(path, []byte("corrupt image"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := clipPreviewBitmap(p, 96); err == nil {
		t.Fatal("corrupt image accepted")
	}
	writeClipImage(t, path, 1, 1)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	binary.BigEndian.PutUint32(data[16:20], 8000)
	binary.BigEndian.PutUint32(data[20:24], 8000)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := clipPreviewBitmap(p, 96); !errors.Is(err, domain.ErrClipOversize) {
		t.Fatalf("pixel bound not enforced before decode: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Truncate(domain.DefaultClipLimits().ItemBytes + 1)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := clipPreviewBitmap(p, 96); !errors.Is(err, domain.ErrClipOversize) {
		t.Fatalf("byte bound not enforced: %v", err)
	}
}

func TestClipboardImageFileHoverDismissalDropsInFlightResult(t *testing.T) {
	a := clipboardFixture(t)
	path := filepath.Join(t.TempDir(), "photo.png")
	writeClipImage(t, path, 1280, 640)
	p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
	item := domain.Clip{ID: p.ID(), Payload: p}
	a.state.Clipboard.Items = []arion.Clip{item}
	testClipboardHoverDismissal(t, a, item)
}

// CLP-A06: optional preview failures neither raise nor replace operation errors.
func TestClipboardPreviewFailuresLeaveHistoryErrorsUnchanged(t *testing.T) {
	for _, existing := range []copy.Key{"", copy.ClipboardPreferencesSaveFailed, copy.ClipboardPasteFailed} {
		t.Run(string(existing), func(t *testing.T) {
			a := clipboardFixture(t)
			corrupt := filepath.Join(t.TempDir(), "corrupt.png")
			if err := os.WriteFile(corrupt, []byte("not an image"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{corrupt, filepath.Join(t.TempDir(), "missing.png")} {
				p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
				if _, err := a.clipboard.service.Capture(p, a.prefs.Clipboard.Limits, time.Now()); err != nil {
					t.Fatal(err)
				}
			}
			a.state.Clipboard.Error = existing
			a.clipboardWork(func() ([]domain.Clip, error) {
				return refreshClipboard(a.clipboard.service, a.prefs.Clipboard.Limits, true)
			}, copy.ClipboardLoadFailed)
			finishClipboardWork(t, a)
			if a.state.Clipboard.Error != existing || len(a.state.Clipboard.Items) != 2 {
				t.Fatal("optional preview failure changed the history error or removed resources")
			}
			for _, meta := range a.state.Clipboard.Meta {
				if meta.Thumbnail != nil {
					t.Fatal("failed thumbnail was published")
				}
			}
		})
	}
}

func TestClipboardFileValidationIsIndependentOfPreview(t *testing.T) {
	a := clipboardFixture(t)
	path := filepath.Join(t.TempDir(), "corrupt.png")
	if err := os.WriteFile(path, []byte("ordinary file with an image extension"), 0600); err != nil {
		t.Fatal(err)
	}
	p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{path}}
	item := domain.Clip{ID: p.ID(), Payload: p}
	a.state.Clipboard.Items = []domain.Clip{item}
	delivered := false
	a.withClipboardResource(item, func() { delivered = true })
	finishClipboardWork(t, a)
	if !delivered || a.state.Clipboard.Error != "" {
		t.Fatal("preview decoding blocked valid file delivery")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	delivered = false
	a.withClipboardResource(item, func() { delivered = true })
	finishClipboardWork(t, a)
	if delivered || a.state.Clipboard.Error != copy.ClipboardUnavailable {
		t.Fatal("silent previews bypassed actual file validation")
	}
}
