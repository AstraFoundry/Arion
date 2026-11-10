package clipboardos

import (
	"errors"
	"io"
	"os"

	"arion.local/core/domain"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

// ReadPreviewFile reads one ordinary file under the same scoped credentials as
// delivery. It never publishes to the clipboard or persists source bytes.
func ReadPreviewFile(p domain.ClipPayload, limit int64) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if p.Kind != domain.ClipFiles || len(p.Files) != 1 || limit <= 0 {
		return nil, domain.ErrClipInvalid
	}
	// Reject known unsupported resources before requesting a legacy scope.
	// Permission-limited metadata must be checked again under the saved grant.
	if info, err := os.Lstat(p.Files[0]); err == nil {
		if !info.Mode().IsRegular() {
			return nil, domain.ErrClipInvalid
		}
	} else if !errors.Is(err, os.ErrPermission) {
		return nil, sourceFileError(err)
	}
	data, err := transfer.FileData(p.Files[0])
	if err != nil {
		return nil, err
	}
	if len(p.FileAccess) != 0 && len(p.FileAccess[0]) != 0 {
		files, err := data.Read(transfer.FileList)
		if err != nil {
			return nil, err
		}
		data = transfer.New(transfer.NewItem(transfer.Bytes(transfer.FileList, files), transfer.Bytes(transfer.MacOSFileBookmark, p.FileAccess[0])))
	}
	var content []byte
	err = mygo.Clipboard.WithFileAccess(data, func() error {
		info, err := os.Lstat(p.Files[0])
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return domain.ErrClipInvalid
		}
		f, err := os.Open(p.Files[0])
		if err != nil {
			return err
		}
		defer f.Close()
		opened, err := f.Stat()
		if err != nil {
			return err
		}
		if !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
			return domain.ErrClipInvalid
		}
		if opened.Size() > limit {
			return domain.ErrClipOversize
		}
		content, err = io.ReadAll(io.LimitReader(f, limit+1))
		if err == nil && int64(len(content)) > limit {
			return domain.ErrClipOversize
		}
		return err
	})
	if err != nil {
		return nil, sourceFileError(err)
	}
	return content, nil
}
