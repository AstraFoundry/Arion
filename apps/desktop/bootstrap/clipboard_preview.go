package bootstrap

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardhistory"
	"arion.local/desktop/infrastructure/clipboardos"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func clipBitmap(data []byte, edge int) (*ui.Bitmap, error) {
	if err := clipboardhistory.ValidateResource(domain.ClipPayload{Kind: domain.ClipImage, PNG: data}); err != nil {
		return nil, err
	}
	return ui.DecodeBitmapThumbnail(data, edge)
}

// File groups keep their ordered names and paths rather than implying that
// one image represents the entire selection. File delivery stays unchanged.
func clipImageFile(p domain.ClipPayload) bool {
	if p.Kind != domain.ClipFiles || len(p.Files) != 1 {
		return false
	}
	switch strings.ToLower(filepath.Ext(p.Files[0])) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return true
	}
	return false
}

func clipPreviewBitmap(p domain.ClipPayload, edge int) (*ui.Bitmap, error) {
	if p.Kind == domain.ClipImage {
		return clipBitmap(p.PNG, edge)
	}
	if !clipImageFile(p) {
		return nil, domain.ErrClipInvalid
	}
	data, err := clipboardos.ReadPreviewFile(p, domain.DefaultClipLimits().ItemBytes)
	if err != nil {
		return nil, err
	}
	if err := clipboardhistory.ValidatePreviewImage(data); err != nil {
		return nil, err
	}
	return ui.DecodeBitmapThumbnail(data, edge)
}

func clipboardMetadata(items []domain.Clip, cached map[string]arion.ClipboardMeta) map[string]arion.ClipboardMeta {
	meta := make(map[string]arion.ClipboardMeta, len(items))
	for _, item := range items {
		m := arion.ClipboardMeta{}
		fileImage := clipImageFile(item.Payload)
		for _, path := range item.Payload.Files {
			f := arion.ClipboardFile{Name: filepath.Base(path), Path: path}
			if info, err := os.Lstat(path); err == nil {
				f.Available = info.Mode().IsRegular()
				f.Directory = info.IsDir()
				fileImage = fileImage && f.Available
				if fileImage {
					version := binary.LittleEndian.AppendUint64(nil, uint64(info.Size()))
					version = binary.LittleEndian.AppendUint64(version, uint64(info.ModTime().UnixNano()))
					if len(item.Payload.FileAccess) != 0 {
						version = append(version, item.Payload.FileAccess[0]...)
					}
					m.ThumbnailRevision = sha256.Sum256(version)
				}
			}
			m.Files = append(m.Files, f)
		}
		if item.Payload.Kind == domain.ClipImage {
			m.Thumbnail = cached[item.ID].Thumbnail
			if m.Thumbnail == nil {
				// CLP-06: preview failures are silent; delivery validates separately.
				m.Thumbnail, _ = clipBitmap(item.Payload.PNG, 96)
			}
		} else if fileImage {
			previous := cached[item.ID]
			if m.ThumbnailRevision != ([32]byte{}) && previous.ThumbnailRevision == m.ThumbnailRevision {
				m.Thumbnail = previous.Thumbnail
			}
			if m.Thumbnail == nil {
				var err error
				m.Thumbnail, err = clipPreviewBitmap(item.Payload, 96)
				if err == nil {
					m.Files[0].Available = true
				}
			}
		}
		meta[item.ID] = m
	}
	return meta
}

// Folder-containing legacy records remain encrypted until retention removes
// them, but cannot appear as supported file resources or be delivered.
func eligibleClipboardItems(items []domain.Clip, meta map[string]arion.ClipboardMeta) []domain.Clip {
	result := make([]domain.Clip, 0, len(items))
	for _, item := range items {
		folder := false
		for _, file := range meta[item.ID].Files {
			folder = folder || file.Directory
		}
		if !folder {
			result = append(result, item)
		}
	}
	return result
}
