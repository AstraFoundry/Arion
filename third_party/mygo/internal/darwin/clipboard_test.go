//go:build darwin

package darwin

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/transfer"
)

func TestClipboardNativeAliasesAndFileList(t *testing.T) {
	load()
	withPool(func() {
		item := autorelease(send(send(class("NSPasteboardItem"), "alloc"), "init"))
		const utf16 = "public.utf16-external-plain-text"
		send(item, "setString:forType:", uintptr(nsString("hello 日本語")), uintptr(nsString(utf16)))
		files := []byte("file:///tmp/a%20%23.txt\r\nfile:///tmp/b.txt")
		send(item, "setData:forType:", uintptr(nsData([]byte("file:///tmp/a%20%23.txt"))), uintptr(nsString("public.file-url")))
		send(item, "setData:forType:", uintptr(nsData(files)), uintptr(nsString("dev.mygo.file-list")))
		var encoded bytes.Buffer
		_ = png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 2, 3)))
		rep := send(class("NSBitmapImageRep"), "imageRepWithData:", uintptr(nsData(encoded.Bytes())))
		tiff := send(rep, "representationUsingType:properties:", 0, uintptr(send(class("NSDictionary"), "dictionary")))
		send(item, "setData:forType:", uintptr(tiff), uintptr(nsString(utTIFF)))
		reps, err := macClipboardReadItem(item, []transfer.Format{transfer.Text, transfer.FileList, transfer.URIList, transfer.PNG, transfer.Text})
		if err != nil {
			t.Fatal(err)
		}
		d := transfer.New(transfer.NewItem(reps...))
		if b, _ := d.Read(transfer.Text); string(b) != "hello 日本語" {
			t.Fatalf("UTF-16 conversion %q", b)
		}
		for _, f := range []transfer.Format{transfer.FileList, transfer.URIList} {
			if b, _ := d.Read(f); !bytes.Equal(b, files) {
				t.Fatalf("complete %s: %q", f, b)
			}
		}
		b, _ := d.Read(transfer.PNG)
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil || img.Bounds() != image.Rect(0, 0, 2, 3) {
			t.Fatalf("TIFF conversion: %v", err)
		}
	})
}

// Finder copies file reference URLs (file:///.file/id=…), which must be
// read as paths.
func TestClipboardFileReferenceURL(t *testing.T) {
	load()
	path := filepath.Join(t.TempDir(), "a b.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(path)
	withPool(func() {
		ref := goString(send(send(fileURL(path), "fileReferenceURL"), "absoluteString"))
		if !strings.HasPrefix(ref, "file:///.file/id=") {
			t.Fatalf("reference URL %q", ref)
		}
		item := autorelease(send(send(class("NSPasteboardItem"), "alloc"), "init"))
		send(item, "setData:forType:", uintptr(nsData([]byte(ref))), uintptr(nsString("public.file-url")))
		for _, f := range []transfer.Format{transfer.FileList, transfer.URIList} {
			reps, err := macClipboardReadItem(item, []transfer.Format{f})
			if err != nil {
				t.Fatal(err)
			}
			b, _ := transfer.New(transfer.NewItem(reps...)).Read(f)
			files, err := transfer.New(transfer.NewItem(transfer.Bytes(transfer.FileList, b))).Files()
			if err != nil || len(files) != 1 || files[0] != want {
				t.Fatalf("%s: files %q, %v; want %q", f, files, err, want)
			}
		}
	})
}
