//go:build darwin

package darwin

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/transfer"
)

func TestClipboardFileURLsPreparedBeforePublication(t *testing.T) {
	load()
	paths := []string{filepath.Join(t.TempDir(), "a 日本語 #.txt"), filepath.Join(t.TempDir(), "b.txt")}
	for _, path := range paths {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := transfer.FileData(paths...)
	if err != nil {
		t.Fatal(err)
	}
	const marker transfer.Format = "application/x-mygo-owned-marker"
	var fileCalls, markerCalls, uriCalls int
	var tagged []transfer.Item
	for _, item := range files.Items() {
		tagged = append(tagged, transfer.NewItem(
			transfer.Lazy(marker, func() ([]byte, error) { markerCalls++; return []byte("owned marker"), nil }),
			transfer.Lazy(transfer.FileList, func() ([]byte, error) { fileCalls++; return item.Read(transfer.FileList) }),
			transfer.Lazy(transfer.URIList, func() ([]byte, error) { uriCalls++; return item.Read(transfer.URIList) }),
		))
	}
	data := transfer.New(tagged...)
	withPool(func() {
		registerClipboardClass()
		s := &macClipboardSource{data: data, items: map[id]transfer.Item{}}
		s.obj = send(send(class("MyGoClipboardProvider"), "alloc"), "init")
		macClipboardProviders[s.obj] = s
		defer s.release()
		items, err := macClipboardItems(data, s.obj, s.items)
		if err != nil {
			t.Fatal(err)
		}
		// This must hold before writeObjects can ask the pasteboard server to
		// publish files. Other representations still keep their lazy lifetime.
		if fileCalls != len(paths) || markerCalls != 0 || uriCalls != 0 {
			t.Fatalf("prepublication calls: files=%d marker=%d URI=%d", fileCalls, markerCalls, uriCalls)
		}
		for n, pb := range items {
			want, _ := files.Items()[n].Read(transfer.FileList)
			if got := goBytes(send(pb, "dataForType:", uintptr(nsString("public.file-url")))); !bytes.Equal(got, want) {
				t.Fatalf("prepared native file %d has %d bytes, want %d", n, len(got), len(want))
			}
		}
		board := send(class("NSPasteboard"), "pasteboardWithUniqueName")
		defer send(board, "releaseGlobally")
		verify := func() {
			t.Helper()
			nativeItems := arrayItems(send(board, "pasteboardItems"))
			decoded, err := macClipboardFileURLs(board, nativeItems, []transfer.Format{transfer.FileList})
			if err != nil || len(decoded) != len(paths) {
				t.Fatalf("native file decoding: count=%d, error=%v", len(decoded), err)
			}
			for n, pb := range nativeItems {
				reps, err := macClipboardReadItem(pb, []transfer.Format{transfer.FileList}, decoded[pb])
				if err != nil {
					t.Fatal(err)
				}
				got, err := transfer.New(transfer.NewItem(reps...)).Files()
				if err != nil || len(got) != 1 || got[0] != paths[n] {
					t.Fatalf("decoded file %d=%q, error=%v", n, got, err)
				}
			}
			urls := arrayItems(send(board, "readObjectsForClasses:options:", uintptr(nsArray(id(class("NSURL")))), 0))
			if len(urls) != len(paths) {
				t.Fatalf("native file URL count=%d, want %d", len(urls), len(paths))
			}
			for n, url := range urls {
				want, _ := filepath.EvalSymlinks(paths[n])
				got, err := filepath.EvalSymlinks(goString(send(send(url, "filePathURL"), "path")))
				if err != nil || got != want {
					t.Errorf("native file %d=%q, want %q", n, got, want)
				}
			}
			for n, pb := range arrayItems(send(board, "pasteboardItems")) {
				want, _ := files.Items()[n].Read(transfer.FileList)
				if got := goBytes(send(pb, "dataForType:", uintptr(nsString("public.file-url")))); !bytes.Equal(got, want) {
					t.Errorf("published native file %d has %d bytes, want %d", n, len(got), len(want))
				}
				if b := goBytes(send(pb, "dataForType:", uintptr(nsString(macType(marker))))); string(b) != "owned marker" {
					t.Errorf("internal marker lost: %q", b)
				}
			}
		}
		if !sendBool(board, "writeObjects:", uintptr(nsArray(items...))) {
			t.Fatal("native publication failed")
		}
		verify()
		persistent, err := data.Materialize(data.Formats())
		if err != nil {
			t.Fatal(err)
		}
		items, err = macClipboardItems(persistent, 0, nil)
		if err != nil {
			t.Fatal(err)
		}
		send(board, "clearContents")
		if !sendBool(board, "writeObjects:", uintptr(nsArray(items...))) {
			t.Fatal("persistent publication failed")
		}
		s.release()
		verify()
		if fileCalls != len(paths) || markerCalls != len(paths) || uriCalls != len(paths) {
			t.Fatalf("provider cache calls: files=%d marker=%d URI=%d", fileCalls, markerCalls, uriCalls)
		}
	})
}

func TestClipboardFileProviderFailureBeforePublication(t *testing.T) {
	load()
	want := errors.New("owned file provider failure")
	data := transfer.New(transfer.NewItem(transfer.Lazy(transfer.FileList, func() ([]byte, error) { return nil, want })))
	withPool(func() {
		registerClipboardClass()
		s := &macClipboardSource{data: data, items: map[id]transfer.Item{}}
		s.obj = send(send(class("MyGoClipboardProvider"), "alloc"), "init")
		macClipboardProviders[s.obj] = s
		defer s.release()
		// A provider requests delayed publication, but file URL
		// preparation must fail synchronously before any item is published.
		items, err := macClipboardItems(data, s.obj, s.items)
		if !errors.Is(err, want) || len(items) != 0 {
			t.Fatalf("items=%d, error=%v", len(items), err)
		}
	})
}

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
		reps, err := macClipboardReadItem(item, []transfer.Format{transfer.Text, transfer.FileList, transfer.URIList, transfer.PNG, transfer.Text}, 0)
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
			reps, err := macClipboardReadItem(item, []transfer.Format{f}, 0)
			if err != nil {
				t.Fatal(err)
			}
			b, _ := transfer.New(transfer.NewItem(reps...)).Read(f)
			files, err := transfer.New(transfer.NewItem(transfer.Bytes(transfer.FileList, b))).Files()
			if err != nil || len(files) != 1 || files[0] != want {
				t.Fatalf("%s: files %q, %v; want %q", f, files, err, want)
			}
		}
		board := send(class("NSPasteboard"), "pasteboardWithUniqueName")
		defer send(board, "releaseGlobally")
		if !sendBool(board, "writeObjects:", uintptr(nsArray(item))) {
			t.Fatal("file reference publication failed")
		}
		nativeItems := arrayItems(send(board, "pasteboardItems"))
		urls, err := macClipboardFileURLs(board, nativeItems, []transfer.Format{transfer.FileList})
		if err != nil || len(urls) != 1 {
			t.Fatalf("typed file reference decoding: count=%d, error=%v", len(urls), err)
		}
		reps, err := macClipboardReadItem(nativeItems[0], []transfer.Format{transfer.FileList}, urls[nativeItems[0]])
		if err != nil {
			t.Fatal(err)
		}
		paths, err := transfer.New(transfer.NewItem(reps...)).Files()
		if err != nil || len(paths) != 1 || paths[0] != want {
			t.Fatalf("typed file reference paths=%q, error=%v; want %q", paths, err, want)
		}
	})
}
