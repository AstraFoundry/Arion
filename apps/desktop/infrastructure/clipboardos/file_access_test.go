package clipboardos

import (
	"arion.local/core/domain"
	"bytes"
	"errors"
	"github.com/egoist/mygo/transfer"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestClipboardFileAccessRequiresSourcePermissionEvidence(t *testing.T) {
	for _, cause := range []error{os.ErrNotExist, errors.New("corrupt bookmark"), os.ErrPermission} {
		original := &os.PathError{Op: "read", Path: filepath.Join(t.TempDir(), "source-file"), Err: cause}
		if FileAccessDenied(original) {
			t.Fatal("arbitrary path failure was classified as source-file access")
		}
		wrapped := sourceFileError(original)
		want := runtime.GOOS == "darwin" && errors.Is(cause, os.ErrPermission)
		if FileAccessDenied(wrapped) != want {
			t.Fatal("incorrect source permission classification", cause)
		}
		var path *os.PathError
		if !errors.As(wrapped, &path) || path != original || !errors.Is(wrapped, cause) {
			t.Fatal("native error identity lost")
		}
		if !want && wrapped != original {
			t.Fatal("non-permission failure was changed")
		}
	}
	profile := &os.PathError{Op: "open", Path: filepath.Join(t.TempDir(), "profile", "clipboard.bin"), Err: os.ErrPermission}
	if FileAccessDenied(profile) || FileAccessDenied(os.ErrPermission) {
		t.Fatal("storage/pathless error became folder-access guidance")
	}
	if FileAccessDenied(sourceFileError(&os.PathError{Op: "read", Path: "relative-file", Err: os.ErrPermission})) {
		t.Fatal("relative provider failure became source evidence")
	}
}

func TestClipboardNativeReadMarksOnlyFilePermissionFailures(t *testing.T) {
	original := &os.PathError{Op: "clipboard bookmark", Path: filepath.Join(t.TempDir(), "source-file"), Err: os.ErrPermission}
	for _, format := range []transfer.Format{transfer.FileList, transfer.Text} {
		_, err := readClipboard([]transfer.Format{format}, true, func(...transfer.Format) (transfer.Data, error) { return transfer.Data{}, original }, 1<<20)
		want := runtime.GOOS == "darwin" && format == transfer.FileList
		if FileAccessDenied(err) != want || !errors.Is(err, os.ErrPermission) {
			t.Fatal("read failure context lost", format, err)
		}
	}
}

// CLP-A08: mixed file copies cannot fall through to text/image capture or grants.
func TestClipboardFilesOffDoesNotReadProvider(t *testing.T) {
	for _, formats := range [][]transfer.Format{{transfer.FileList}, {transfer.PNG, transfer.FileList, transfer.Text}} {
		_, err := readClipboard(formats, false, func(...transfer.Format) (transfer.Data, error) {
			t.Fatal("disabled file provider read")
			return transfer.Data{}, nil
		}, 1<<20)
		if !errors.Is(err, transfer.ErrFormat) {
			t.Fatal("disabled files accepted", err)
		}
	}
	data, err := transfer.FileData(filepath.Join(t.TempDir(), "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = readClipboard([]transfer.Format{transfer.URIList}, false, func(formats ...transfer.Format) (transfer.Data, error) {
		for _, f := range formats {
			if f == transfer.MacOSFileBookmark {
				t.Fatal("disabled URI requested a bookmark")
			}
		}
		return data, nil
	}, 1<<20)
	if !errors.Is(err, transfer.ErrFormat) {
		t.Fatal("URI-only files bypassed opt-in", err)
	}
	text := transfer.TextData("kept text")
	payload, err := readClipboard(text.Formats(), false, func(...transfer.Format) (transfer.Data, error) { return text, nil }, 1<<20)
	if err != nil || payload.Kind != domain.ClipText {
		t.Fatal("text affected by file preference", err)
	}
}

func TestClipboardFilesOffPreservesDirectImageBytes(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	data := transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, encoded.Bytes())))
	payload, err := readClipboard(data.Formats(), false, func(...transfer.Format) (transfer.Data, error) { return data, nil }, 1<<20)
	if err != nil || payload.Kind != domain.ClipImage || !bytes.Equal(payload.PNG, encoded.Bytes()) {
		t.Fatal("direct bitmap capture affected by disabled files", err)
	}
}

func TestClipboardFilesOffRejectsURIFileAlternativesBeforeFallbacks(t *testing.T) {
	for _, raw := range []string{"file:///owned/file.txt", "FILE:///owned/file.txt", "file://server/shared/file.txt"} {
		data := transfer.New(transfer.NewItem(transfer.Bytes(transfer.URIList, []byte(raw)), transfer.Bytes(transfer.Text, []byte(raw)), transfer.Bytes(transfer.PNG, []byte("image alternative"))))
		reads := 0
		_, err := readClipboard(data.Formats(), false, func(formats ...transfer.Format) (transfer.Data, error) {
			reads++
			if len(formats) != 1 || formats[0] != transfer.URIList {
				t.Fatal("file alternative was read before URI exclusion", formats)
			}
			return data.Materialize(formats)
		}, 1<<20)
		if !errors.Is(err, transfer.ErrFormat) || reads != 1 {
			t.Fatal("URI file leaked through another representation", raw, err, reads)
		}
	}
}

func TestClipboardInternalURIIsNotInspected(t *testing.T) {
	_, err := readClipboard([]transfer.Format{Internal, transfer.URIList}, false, func(...transfer.Format) (transfer.Data, error) {
		t.Fatal("internal URI provider was read")
		return transfer.Data{}, nil
	}, 1<<20)
	if !errors.Is(err, transfer.ErrFormat) {
		t.Fatal("internal offer accepted", err)
	}
}

func TestClipboardFilesOffPreservesNonFileURIOffers(t *testing.T) {
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []domain.ClipKind{domain.ClipLink, domain.ClipImage} {
		reps := []transfer.Representation{transfer.Bytes(transfer.URIList, []byte("https://example.com/resource")), transfer.Bytes(transfer.Text, []byte("https://example.com/resource"))}
		if kind == domain.ClipImage {
			reps = append(reps, transfer.Bytes(transfer.PNG, encoded.Bytes()))
		}
		data := transfer.New(transfer.NewItem(reps...))
		payload, err := readClipboard(data.Formats(), false, func(formats ...transfer.Format) (transfer.Data, error) {
			for _, format := range formats {
				if format == transfer.MacOSFileBookmark {
					t.Fatal("non-file inspection requested native file access")
				}
			}
			return data.Materialize(formats)
		}, 1<<20)
		if err != nil || payload.Kind != kind {
			t.Fatal("non-file representation changed", kind, err)
		}
		if kind == domain.ClipImage && !bytes.Equal(payload.PNG, encoded.Bytes()) {
			t.Fatal("image bytes changed")
		}
	}
	data := transfer.New(transfer.NewItem(transfer.Bytes(transfer.URIList, []byte("https://example.com/only-uri"))))
	reads := 0
	payload, err := readClipboard(data.Formats(), false, func(formats ...transfer.Format) (transfer.Data, error) { reads++; return data.Materialize(formats) }, 1<<20)
	if err != nil || payload.Kind != domain.ClipLink || reads != 1 {
		t.Fatal("URI-only link was re-read or changed", err, reads)
	}
	unavailable := errors.New("URI provider unavailable")
	_, err = readClipboard([]transfer.Format{transfer.URIList, transfer.PNG}, false, func(formats ...transfer.Format) (transfer.Data, error) {
		if len(formats) != 1 || formats[0] != transfer.URIList {
			t.Fatal("provider failure silently fell through to another representation")
		}
		return transfer.Data{}, unavailable
	}, 1<<20)
	if err != unavailable || FileAccessDenied(err) {
		t.Fatal("provider failure was replaced", err)
	}
}
