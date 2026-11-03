//go:build darwin

package darwin

import (
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/transfer"
)

var (
	clipboardClassOnce    sync.Once
	macClipboard          *macClipboardSource
	macClipboardProviders = map[id]*macClipboardSource{}
)

type macClipboardSource struct {
	obj      id
	change   int
	data     transfer.Data
	items    map[id]transfer.Item
	released func()
}

func registerClipboardClass() {
	clipboardClassOnce.Do(func() {
		classDef("MyGoClipboardProvider", "NSObject", []string{"NSPasteboardItemDataProvider"}, []objc.MethodDef{
			method("pasteboard:item:provideDataForType:", func(self id, _ objc.SEL, board, item, typ id) {
				s := macClipboardProviders[self]
				if s == nil {
					return
				}
				name := goString(typ)
				b, err := macClipboardBytes(s.items[item], name)
				if err != nil {
					return
				}
				send(item, "setData:forType:", uintptr(nsData(b)), uintptr(typ))
			}),
			method("pasteboardFinishedWithDataProvider:", func(self id, _ objc.SEL, board id) {
				if s := macClipboardProviders[self]; s != nil {
					s.release()
				}
			}),
		})
	})
}

func (s *macClipboardSource) release() {
	if s.obj == 0 {
		return
	}
	delete(macClipboardProviders, s.obj)
	if macClipboard == s {
		macClipboard = nil
	}
	// AppKit may still be in a callback on self; its own references are
	// independent of our +1. Drop all Go closures before notifying the core.
	autorelease(s.obj)
	s.obj, s.data, s.items = 0, transfer.Data{}, nil
	if fn := s.released; fn != nil {
		s.released = nil
		fn()
	}
}

func currentMacClipboard() *macClipboardSource {
	s := macClipboard
	if s != nil && sendInt(pasteboard(), "changeCount") != s.change {
		s.release()
		return nil
	}
	return s
}

// A file-list representation can contain several URLs. Keep its complete
// bytes alongside AppKit's single-file URL interoperability representation.
func macClipboardFormat(name string) transfer.Format {
	if name == "public.utf16-plain-text" || name == "public.utf16-external-plain-text" {
		return transfer.Text
	}
	if name == "dev.mygo.file-list" || name == "NSFilenamesPboardType" {
		return transfer.FileList
	}
	return macFormat(name)
}
func macClipboardTypes(item transfer.Item) []string {
	var types []string
	for _, f := range item.Formats() {
		if f == transfer.MacOSFileBookmark {
			continue
		}
		types = append(types, macType(f))
		if f == transfer.URIList {
			types = append(types, "public.url")
		}
		if f == transfer.FileList {
			types = append(types, "dev.mygo.file-list")
		}
	}
	return types
}

func macClipboardBytes(item transfer.Item, name string) ([]byte, error) {
	b, err := item.Read(macClipboardFormat(name))
	if err != nil {
		return nil, err
	}
	if name == "public.url" || name == "public.file-url" {
		b = []byte(strings.TrimSpace(strings.Split(string(b), "\n")[0]))
	}
	return b, nil
}

// A zero provider builds persistent items. File URLs must always be eager:
// the pasteboard server needs their bytes during publication to create file
// access extensions, before it can request delayed representations.
func macClipboardItems(data transfer.Data, provider id, provided map[id]transfer.Item) ([]id, error) {
	var items []id
	for _, item := range data.Items() {
		pb := autorelease(send(send(class("NSPasteboardItem"), "alloc"), "init"))
		var delayed []id
		for _, name := range macClipboardTypes(item) {
			typ := nsString(name)
			if provider != 0 && name != "public.file-url" {
				delayed = append(delayed, typ)
				continue
			}
			b, err := macClipboardBytes(item, name)
			if err != nil {
				return nil, err
			}
			if !sendBool(pb, "setData:forType:", uintptr(nsData(b)), uintptr(typ)) {
				return nil, errors.New("mygo: cannot prepare pasteboard representation")
			}
		}
		if len(delayed) != 0 {
			provided[pb] = item
			if !sendBool(pb, "setDataProvider:forTypes:", uintptr(provider), uintptr(nsArray(delayed...))) {
				return nil, errors.New("mygo: cannot prepare pasteboard provider")
			}
		}
		items = append(items, pb)
	}
	return items, nil
}

func macClipboardFileURLs(board id, items []id, formats []transfer.Format) (map[id]id, error) {
	if !slices.Contains(formats, transfer.FileList) && !slices.Contains(formats, transfer.URIList) && !slices.Contains(formats, transfer.MacOSFileBookmark) {
		return nil, nil
	}
	var fileItems []id
	for _, item := range items {
		for _, typ := range arrayItems(send(item, "types")) {
			if goString(typ) == "public.file-url" {
				fileItems = append(fileItems, item)
				break
			}
		}
	}
	if len(fileItems) == 0 {
		return nil, nil
	}
	// AppKit consumes the pasteboard's file-access extensions when decoding
	// NSURLs. Parsing dataForType bytes alone loses this access, even when
	// the resulting path can still be inspected with stat.
	options := send(class("NSDictionary"), "dictionaryWithObject:forKey:", uintptr(nsBool(true)), uintptr(constString(libAppKit, "NSPasteboardURLReadingFileURLsOnlyKey")))
	urls := arrayItems(send(board, "readObjectsForClasses:options:", uintptr(nsArray(id(class("NSURL")))), uintptr(options)))
	if len(urls) != len(fileItems) {
		return nil, errors.New("mygo: cannot read native clipboard file URLs")
	}
	result := make(map[id]id, len(urls))
	for n, url := range urls {
		if !sendBool(url, "isFileURL") || send(url, "filePathURL") == 0 {
			return nil, errors.New("mygo: invalid native clipboard file URL")
		}
		result[fileItems[n]] = url
	}
	return result, nil
}

func (clipboard) WriteData(data transfer.Data, released func()) error {
	var err error
	withPool(func() {
		registerClipboardClass()
		if len(data.Formats()) == 0 {
			clipboard{}.Clear()
			if released != nil {
				released()
			}
			return
		}
		closeAccess, e := macClipboardFileAccess(data)
		if e != nil {
			err = e
			return
		}
		defer closeAccess()
		s := &macClipboardSource{data: data, items: map[id]transfer.Item{}, released: released}
		s.obj = send(send(class("MyGoClipboardProvider"), "alloc"), "init")
		macClipboardProviders[s.obj] = s
		items, e := macClipboardItems(data, s.obj, s.items)
		if e != nil {
			s.released = nil
			s.release()
			err = e
			return
		}
		board := pasteboard()
		send(board, "clearContents")
		if old := macClipboard; old != nil {
			old.release()
		}
		if !sendBool(board, "writeObjects:", uintptr(nsArray(items...))) {
			s.released = nil
			s.release()
			err = errors.New("mygo: cannot write pasteboard items")
			return
		}
		s.change = sendInt(board, "changeCount")
		if s.obj != 0 {
			macClipboard = s
		}
	})
	return err
}

func (clipboard) Formats() []transfer.Format {
	var out []transfer.Format
	withPool(func() {
		if s := currentMacClipboard(); s != nil {
			out = s.data.Formats()
			return
		}
		for _, typ := range arrayItems(send(pasteboard(), "types")) {
			name := goString(typ)
			f := macClipboardFormat(name)
			if name == utTIFF {
				f = transfer.PNG
			}
			if f == "" || strings.ContainsAny(string(f), "\x00\r\n") || f == platform.DragSessionFormat || f == transfer.MacOSFileBookmark {
				continue
			}
			if !slices.Contains(out, f) {
				out = append(out, f)
			}
			if f == transfer.FileList && !slices.Contains(out, transfer.URIList) {
				out = append(out, transfer.URIList)
			}
		}
	})
	return out
}

func (c clipboard) ReadData(formats []transfer.Format) (transfer.Data, error) {
	var result transfer.Data
	var err error
	withPool(func() {
		board := pasteboard()
		change := sendInt(board, "changeCount")
		if len(formats) == 0 {
			formats = c.Formats()
			if len(formats) == 0 {
				return
			}
		}
		if s := currentMacClipboard(); s != nil {
			result, err = s.data.Materialize(formats)
		} else {
			var items []transfer.Item
			nativeItems := arrayItems(send(board, "pasteboardItems"))
			fileURLs, e := macClipboardFileURLs(board, nativeItems, formats)
			if e != nil {
				err = e
				return
			}
			for _, item := range nativeItems {
				reps, e := macClipboardReadItem(item, formats, fileURLs[item])
				if e != nil {
					err = e
					return
				}
				if len(reps) != 0 {
					items = append(items, transfer.NewItem(reps...))
				}
			}
			result = transfer.New(items...)
			if len(result.Formats()) == 0 {
				err = transfer.ErrFormat
			}
		}
		if sendInt(board, "changeCount") != change {
			result, err = transfer.Data{}, platform.ErrClipboardChanged
		}
	})
	return result, err
}

func macClipboardReadItem(item id, formats []transfer.Format, fileURL id) ([]transfer.Representation, error) {
	available := map[transfer.Format][]string{}
	for _, typ := range arrayItems(send(item, "types")) {
		name := goString(typ)
		f := macClipboardFormat(name)
		if name == utTIFF {
			f = transfer.PNG
		}
		if f == "" || f == platform.DragSessionFormat || strings.ContainsAny(string(f), "\x00\r\n") {
			continue
		}
		available[f] = append(available[f], name)
		if f == transfer.FileList {
			available[transfer.URIList] = append(available[transfer.URIList], name)
		}
	}
	var reps []transfer.Representation
	for n, f := range formats {
		if slices.Contains(formats[:n], f) {
			continue
		}
		if f == transfer.MacOSFileBookmark {
			if fileURL != 0 {
				b, err := macClipboardBookmark(fileURL)
				if err != nil {
					return nil, err
				}
				reps = append(reps, transfer.Bytes(f, b))
			}
			continue
		}
		names := available[f]
		// Put full serialized URI/file lists before single URL fallbacks.
		preferred := []string{macType(f)}
		if f == transfer.FileList {
			preferred = []string{"dev.mygo.file-list", macType(f)}
		}
		if f == transfer.URIList {
			preferred = []string{macType(f), "dev.mygo.file-list"}
		}
		var ordered []string
		for _, name := range append(preferred, names...) {
			if slices.Contains(names, name) && !slices.Contains(ordered, name) {
				ordered = append(ordered, name)
			}
		}
		var b []byte
		found := false
		for _, name := range ordered {
			typ := nsString(name)
			if name == "public.file-url" && fileURL != 0 {
				path := goString(send(send(fileURL, "filePathURL"), "path"))
				files, err := transfer.FileData(path)
				if err != nil {
					return nil, err
				}
				b, err = files.Read(f)
				if err != nil {
					return nil, err
				}
				found = true
				break
			}
			if name == "NSFilenamesPboardType" {
				var paths []string
				for _, value := range arrayItems(send(item, "propertyListForType:", uintptr(typ))) {
					paths = append(paths, goString(value))
				}
				files, err := transfer.FileData(paths...)
				if err != nil {
					return nil, err
				}
				b, err = files.Read(f)
				if err != nil {
					continue
				}
				found = true
				break
			}
			value := send(item, "dataForType:", uintptr(typ))
			if value == 0 {
				continue
			}
			if send(value, "length") > 64<<20 {
				return nil, errors.New("mygo: clipboard representation exceeds 64 MiB")
			}
			b = goBytes(value)
			if name == "public.file-url" {
				b = filePathURLBytes(b)
			}
			if f == transfer.Text && name != utString {
				// AppKit decodes legacy/UTF-16 text into an NSString.
				b = []byte(goString(send(item, "stringForType:", uintptr(typ))))
			}
			if f == transfer.PNG && name == utTIFF {
				rep := send(class("NSBitmapImageRep"), "imageRepWithData:", uintptr(value))
				if rep == 0 {
					return nil, errors.New("mygo: cannot convert clipboard image")
				}
				b = goBytes(send(rep, "representationUsingType:properties:", 4, uintptr(send(class("NSDictionary"), "dictionary"))))
			}
			found = true
			break
		}
		if found {
			reps = append(reps, transfer.Bytes(f, b))
		}
	}
	return reps, nil
}

func (clipboard) Flush() error {
	var err error
	withPool(func() {
		s := currentMacClipboard()
		if s == nil {
			return
		}
		d, e := s.data.Materialize(s.data.Formats())
		if e != nil {
			err = e
			return
		}
		if currentMacClipboard() != s {
			err = platform.ErrClipboardChanged
			return
		}
		closeAccess, e := macClipboardFileAccess(d)
		if e != nil {
			err = e
			return
		}
		defer closeAccess()
		items, e := macClipboardItems(d, 0, nil)
		if e != nil {
			err = e
			return
		}
		board := pasteboard()
		send(board, "clearContents")
		if !sendBool(board, "writeObjects:", uintptr(nsArray(items...))) {
			err = errors.New("mygo: cannot persist pasteboard items")
		}
		s.release()
	})
	return err
}

func (clipboard) Close() {
	withPool(func() {
		if s := currentMacClipboard(); s != nil {
			send(pasteboard(), "clearContents")
			s.release()
		}
	})
}
