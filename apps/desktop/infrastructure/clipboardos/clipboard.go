// Package clipboardos adapts typed history resources to MyGo's native transfer
// model. Transfers run on the main thread; Sequence is a counter-only observer
// safe on the polling worker.
package clipboardos

import (
	"bytes"
	"crypto/rand"
	"errors"
	"os"
	"slices"
	"strings"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/browser"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

const Internal transfer.Format = "application/x-arion-internal"

var ErrBusy = errors.New("clipboard transaction in progress")

type Target interface {
	// Anchor is the captured input geometry in top-left screen DIPs.
	Anchor() (x, y, width, height int, ok bool)
	Activate() error
	Ready() bool
	Check() error
	Paste(uint64) error
	Close()
}

func Read(limit int64) (domain.ClipPayload, error) {
	formats := mygo.Clipboard.Formats()
	format, err := captureFormat(formats)
	if err != nil {
		return domain.ClipPayload{}, err
	}
	data, err := mygo.Clipboard.Read(format)
	if err != nil {
		return domain.ClipPayload{}, err
	}
	return capturePayload(data, format, limit)
}

func captureFormat(formats []transfer.Format) (transfer.Format, error) {
	// MyGo maps custom MIME formats to platform type identifiers on macOS.
	// Check the portable marker too, after native revision/sensitive filtering.
	if slices.Contains(formats, Internal) {
		return "", transfer.ErrFormat
	}
	var format transfer.Format
	switch {
	case slices.Contains(formats, transfer.FileList):
		format = transfer.FileList
	case slices.Contains(formats, transfer.PNG):
		format = transfer.PNG
	case slices.Contains(formats, transfer.Text):
		format = transfer.Text
	case slices.Contains(formats, transfer.URIList):
		format = transfer.URIList
	default:
		return "", transfer.ErrFormat
	}
	return format, nil
}

func capturePayload(data transfer.Data, format transfer.Format, limit int64) (domain.ClipPayload, error) {
	b, err := data.Read(format)
	if err != nil {
		return domain.ClipPayload{}, err
	}
	if int64(len(b)) > limit {
		return domain.ClipPayload{}, domain.ErrClipOversize
	}
	var p domain.ClipPayload
	switch format {
	case transfer.FileList:
		files, err := data.Files()
		if err != nil {
			return p, err
		}
		p = domain.ClipPayload{Kind: domain.ClipFiles, Files: files}
	case transfer.PNG:
		p = domain.ClipPayload{Kind: domain.ClipImage, PNG: b}
	case transfer.Text:
		p = textPayload(string(b))
	case transfer.URIList:
		files, err := data.Files()
		if err != nil {
			return p, err
		}
		if len(files) > 0 {
			p = domain.ClipPayload{Kind: domain.ClipFiles, Files: files}
		} else {
			urls, err := data.URLs()
			if err != nil || len(urls) != 1 {
				return p, transfer.ErrFormat
			}
			p = textPayload(urls[0])
		}
	}
	return p, p.Validate()
}

func textPayload(text string) domain.ClipPayload {
	p := domain.ClipTextPayload(text)
	if !strings.ContainsAny(text, " \r\n\t") && browser.Validate(text) == nil {
		p.Kind = domain.ClipLink
	}
	return p
}

// FilesOnly excludes folders without changing the order of ordinary files.
// It runs on a resource worker because paths may be on mounted filesystems.
func FilesOnly(p domain.ClipPayload) (domain.ClipPayload, error) {
	if p.Kind != domain.ClipFiles {
		return p, nil
	}
	files := make([]string, 0, len(p.Files))
	for _, path := range p.Files {
		info, err := os.Stat(path)
		if err != nil {
			return domain.ClipPayload{}, err
		}
		if info.Mode().IsRegular() {
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		return domain.ClipPayload{}, transfer.ErrFormat
	}
	p.Files = files
	return p, p.Validate()
}

func ValidateFiles(p domain.ClipPayload) error {
	supported, err := FilesOnly(p)
	if err != nil {
		return err
	}
	if len(supported.Files) != len(p.Files) {
		return transfer.ErrFormat
	}
	return nil
}

// Write tags every item. A copy survives closing the history window and is
// materialized before a paste so no delayed provider depends on that window.
func Write(p domain.ClipPayload) (uint64, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}
	var data transfer.Data
	var err error
	switch p.Kind {
	case domain.ClipFiles:
		data, err = transfer.FileData(p.Files...)
	case domain.ClipImage:
		data = transfer.New(transfer.NewItem(transfer.Bytes(transfer.PNG, p.PNG)))
	default:
		data = transfer.TextData(p.Text)
	}
	if err != nil {
		return 0, err
	}
	marker := make([]byte, 16)
	if _, err := rand.Read(marker); err != nil {
		return 0, err
	}
	var items []transfer.Item
	for _, item := range data.Items() {
		reps := []transfer.Representation{transfer.Bytes(Internal, marker)}
		for _, f := range item.Formats() {
			b, err := item.Read(f)
			if err != nil {
				return 0, err
			}
			reps = append(reps, transfer.Bytes(f, b))
		}
		items = append(items, transfer.NewItem(reps...))
	}
	if err := mygo.Clipboard.Write(transfer.New(items...)); err != nil {
		return 0, err
	}
	if err := mygo.Clipboard.Flush(); err != nil {
		return 0, err
	}
	// A fresh revision alone cannot prove ownership: another app may have
	// copied between Write and this read. Bind it to this exact operation's
	// random marker, then require the native post to match that revision.
	revision := Sequence()
	data, err = mygo.Clipboard.Read(Internal)
	if err != nil {
		return 0, err
	}
	actual, err := data.Read(Internal)
	if err != nil {
		return 0, err
	}
	if !bytes.Equal(marker, actual) || Sequence() != revision {
		return 0, mygo.ErrClipboardChanged
	}
	return revision, nil
}
