//go:build darwin

package darwin

import (
	"errors"
	"slices"
	"unsafe"

	"github.com/egoist/mygo/transfer"
)

const (
	bookmarkReadOnlyScope = 1<<11 | 1<<12
	bookmarkResolveScope  = 1<<8 | 1<<9 | 1<<10 | 1<<15
)

func macClipboardBookmark(url id) ([]byte, error) {
	if sendBool(url, "startAccessingSecurityScopedResource") {
		defer send(url, "stopAccessingSecurityScopedResource")
	}
	data := send(url, "bookmarkDataWithOptions:includingResourceValuesForKeys:relativeToURL:error:", bookmarkReadOnlyScope, 0, 0, 0)
	if data == 0 {
		return nil, errors.New("mygo: cannot preserve clipboard file access")
	}
	return goBytes(data), nil
}

// Keep each file's scope active until the pasteboard server has published its
// receiving-process extensions. Bookmark bytes stay inside Data, never on the
// native pasteboard. Every successful start has one matching stop.
func macClipboardFileAccess(data transfer.Data) (func(), error) {
	var active []id
	close := func() {
		for _, url := range active {
			send(url, "stopAccessingSecurityScopedResource")
		}
		active = nil
	}
	fail := func(err error) (func(), error) { close(); return nil, err }
	for _, item := range data.Items() {
		if !slices.Contains(item.Formats(), transfer.FileList) {
			continue
		}
		paths, err := transfer.New(item).Files()
		if err != nil || len(paths) == 0 {
			return fail(errors.New("mygo: invalid clipboard file references"))
		}
		bookmark, err := item.Read(transfer.MacOSFileBookmark)
		if err != nil && !errors.Is(err, transfer.ErrFormat) {
			return fail(err)
		}
		if len(bookmark) != 0 && len(paths) != 1 {
			return fail(errors.New("mygo: a file bookmark requires one file per item"))
		}
		for _, path := range paths {
			b := bookmark
			if len(b) == 0 {
				// Legacy path-only references still require current access. Failure
				// does not authorize a prompt, broad grant or unscoped fallback.
				b, err = macClipboardBookmark(fileURL(path))
				if err != nil {
					return fail(err)
				}
			}
			var stale byte
			url := send(class("NSURL"), "URLByResolvingBookmarkData:options:relativeToURL:bookmarkDataIsStale:error:", uintptr(nsData(b)), bookmarkResolveScope, 0, uintptr(unsafe.Pointer(&stale)), 0)
			if url == 0 || stale != 0 || !sendBool(url, "isFileURL") {
				return fail(errors.New("mygo: clipboard file access is unavailable"))
			}
			if !sendBool(url, "startAccessingSecurityScopedResource") {
				return fail(errors.New("mygo: cannot activate clipboard file access"))
			}
			active = append(active, url)
			original := goString(send(send(fileURL(path), "URLByResolvingSymlinksInPath"), "path"))
			resolved := goString(send(send(url, "URLByResolvingSymlinksInPath"), "path"))
			if original == "" || resolved != original {
				return fail(errors.New("mygo: clipboard file destination changed"))
			}
		}
	}
	return close, nil
}
