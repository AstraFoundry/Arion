//go:build darwin

package darwin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/egoist/mygo/transfer"
)

type ownedFileAccess struct {
	Path     string
	Bookmark []byte
}

func TestClipboardPermissionErrorClassification(t *testing.T) {
	load()
	withPool(func() {
		posix := send(class("NSError"), "errorWithDomain:code:userInfo:", uintptr(nsString("NSPOSIXErrorDomain")), 1, 0)
		info := send(class("NSDictionary"), "dictionaryWithObject:forKey:", uintptr(posix), uintptr(nsString("NSUnderlyingError")))
		wrapped := send(class("NSError"), "errorWithDomain:code:userInfo:", uintptr(nsString("NSCocoaErrorDomain")), 256, uintptr(info))
		if !macClipboardPermissionError(wrapped) {
			t.Fatal("underlying permission evidence discarded")
		}
		missing := send(class("NSError"), "errorWithDomain:code:userInfo:", uintptr(nsString("NSPOSIXErrorDomain")), 2, 0)
		if macClipboardPermissionError(missing) {
			t.Fatal("missing file classified as permission denial")
		}
		path := filepath.Join(t.TempDir(), "missing")
		_, err := macClipboardBookmark(fileURL(path))
		if err == nil || errors.Is(err, os.ErrPermission) {
			t.Fatal("missing file lost its failure classification", err)
		}
	})
}

func TestClipboardHistorySkipsUnsupportedBookmarks(t *testing.T) {
	load()
	withPool(func() {
		dir := t.TempDir()
		file := filepath.Join(dir, "file")
		if err := os.WriteFile(file, nil, 0600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(file, link); err != nil {
			t.Fatal(err)
		}
		for _, path := range []string{dir, link} {
			bookmark, err := macClipboardOrdinaryFileBookmark(fileURL(path))
			if err != nil || len(bookmark) != 0 {
				t.Fatal("unsupported resource requested a history credential", err)
			}
		}
	})
}

func ownedFileAccessData(path string, bookmark []byte) transfer.Data {
	files, _ := transfer.FileData(path)
	b, _ := files.Read(transfer.FileList)
	return transfer.New(transfer.NewItem(transfer.Bytes(transfer.FileList, b), transfer.Bytes(transfer.MacOSFileBookmark, bookmark)))
}

func TestClipboardFileAccessSurvivesProcessRestart(t *testing.T) {
	load()
	if os.Getenv("MYGO_OWNED_FILE_ACCESS_CHILD") == "1" {
		var input ownedFileAccess
		if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
			t.Fatal(err)
		}
		withPool(func() {
			data := ownedFileAccessData(input.Path, input.Bookmark)
			closeAccess, err := macClipboardFileAccess(data)
			if err != nil {
				t.Fatal(err)
			}
			defer closeAccess()
			items, err := macClipboardItems(data, 0, nil)
			if err != nil {
				t.Fatal(err)
			}
			board := send(class("NSPasteboard"), "pasteboardWithUniqueName")
			defer send(board, "releaseGlobally")
			if !sendBool(board, "writeObjects:", uintptr(nsArray(items...))) {
				t.Fatal("owned publication failed")
			}
			nativeItems := arrayItems(send(board, "pasteboardItems"))
			if len(nativeItems) != 1 {
				t.Fatal("owned file item did not survive publication")
			}
			expected, _ := data.Read(transfer.FileList)
			for _, item := range nativeItems {
				actual := goBytes(send(item, "dataForType:", uintptr(nsString("public.file-url"))))
				if !bytes.Equal(actual, expected) {
					t.Fatal("standard file URL was not published")
				}
				for _, typ := range arrayItems(send(item, "types")) {
					if macClipboardFormat(goString(typ)) == transfer.MacOSFileBookmark {
						t.Fatal("native bookmark credential was published")
					}
				}
			}
			closeAccess()
			fmt.Println("owned_file_scope_verified")
		})
		return
	}
	path := filepath.Join(t.TempDir(), "owned 日本語 #.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var bookmark []byte
	withPool(func() {
		board := send(class("NSPasteboard"), "pasteboardWithUniqueName")
		defer send(board, "releaseGlobally")
		url := send(fileURL(path), "fileReferenceURL")
		if !sendBool(board, "writeObjects:", uintptr(nsArray(url))) {
			t.Fatal("owned capture fixture publication failed")
		}
		items := arrayItems(send(board, "pasteboardItems"))
		formats := []transfer.Format{transfer.FileList, transfer.MacOSFileBookmark}
		urls, err := macClipboardFileURLs(board, items, formats)
		if err != nil {
			t.Fatal(err)
		}
		reps, err := macClipboardReadItem(items[0], formats, urls[items[0]])
		if err != nil {
			t.Fatal(err)
		}
		data := transfer.New(transfer.NewItem(reps...))
		bookmark, err = data.Read(transfer.MacOSFileBookmark)
		if err != nil || len(bookmark) == 0 {
			t.Fatal("native file access was not captured")
		}
		if slices.Contains(macClipboardTypes(data.Items()[0]), macType(transfer.MacOSFileBookmark)) {
			t.Fatal("native bookmark was advertised for publication")
		}
	})
	// The source NSURL and its pool are gone. Only serialized reference
	// credentials cross into a fresh process; the system clipboard is unused.
	input, _ := json.Marshal(ownedFileAccess{Path: path, Bookmark: bookmark})
	cmd := exec.Command(os.Args[0], "-test.run=^TestClipboardFileAccessSurvivesProcessRestart$")
	cmd.Env = append(os.Environ(), "MYGO_OWNED_FILE_ACCESS_CHILD=1")
	cmd.Stdin = bytes.NewReader(input)
	out, err := cmd.CombinedOutput()
	if err != nil || !bytes.Contains(out, []byte("owned_file_scope_verified")) {
		t.Fatalf("owned file access child: %v; %s", err, out)
	}
}

func TestClipboardFileAccessRejectsWrongMovedAndInvalidReferences(t *testing.T) {
	load()
	dir := t.TempDir()
	first, second := filepath.Join(dir, "first.txt"), filepath.Join(dir, "second.txt")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	withPool(func() {
		bookmark, err := macClipboardBookmark(fileURL(first))
		if err != nil {
			t.Fatal(err)
		}
		for _, data := range []transfer.Data{ownedFileAccessData(second, bookmark), ownedFileAccessData(first, []byte("invalid owned bookmark"))} {
			closeAccess, err := macClipboardFileAccess(data)
			if closeAccess != nil {
				closeAccess()
			}
			if err == nil {
				t.Fatal("unbound or invalid native access accepted")
			}
		}
		if err := os.Rename(first, filepath.Join(dir, "moved.txt")); err != nil {
			t.Fatal(err)
		}
		closeAccess, err := macClipboardFileAccess(ownedFileAccessData(first, bookmark))
		if closeAccess != nil {
			closeAccess()
		}
		if err == nil {
			t.Fatal("bookmark followed a moved file")
		}
	})
}

func TestClipboardWithFileAccessCallbackAndFailure(t *testing.T) {
	load()
	path := filepath.Join(t.TempDir(), "owned-preview.png")
	content := []byte("owned preview source")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	var bookmark []byte
	withPool(func() {
		var err error
		bookmark, err = macClipboardOrdinaryFileBookmark(fileURL(path))
		if err != nil {
			t.Fatal(err)
		}
	})
	data := ownedFileAccessData(path, bookmark)
	expected := errors.New("owned read failure")
	err := (clipboard{}).WithFileAccess(data, func() error {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, content) {
			t.Fatal("scope did not allow reading", err)
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatal("callback failure discarded", err)
	}
	calls := 0
	for _, bad := range []transfer.Data{
		ownedFileAccessData(path, []byte("invalid grant")),
		ownedFileAccessData(path+".other", bookmark),
	} {
		if err := (clipboard{}).WithFileAccess(bad, func() error { calls++; return nil }); err == nil {
			t.Fatal("invalid reference authorized callback")
		}
	}
	if calls != 0 {
		t.Fatal("invalid scope ran callback")
	}
	func() {
		defer func() {
			if recover() != expected {
				t.Error("callback panic was swallowed")
			}
		}()
		_ = (clipboard{}).WithFileAccess(data, func() error { panic(expected) })
	}()
	if err := (clipboard{}).WithFileAccess(data, func() error { return nil }); err != nil {
		t.Fatal("scope unusable after failed callback", err)
	}
}
