//go:build darwin

package darwin

import (
	"bytes"
	"encoding/json"
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
