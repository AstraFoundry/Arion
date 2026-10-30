package bootstrap

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/clipboardhistory"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
)

func clipBitmap(data []byte, edge int) (*ui.Bitmap, error) {
	if err := clipboardhistory.ValidateResource(domain.ClipPayload{Kind: domain.ClipImage, PNG: data}); err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if max(w, h) > edge {
		if w >= h {
			h = max(1, h*edge/w)
			w = edge
		} else {
			w = max(1, w*edge/h)
			h = edge
		}
		resized := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.ApproxBiLinear.Scale(resized, resized.Bounds(), img, b, draw.Src, nil)
		img = resized
	}
	return ui.NewBitmap(img), nil
}

func clipboardMetadata(items []domain.Clip, cached map[string]arion.ClipboardMeta) (map[string]arion.ClipboardMeta, error) {
	meta := make(map[string]arion.ClipboardMeta, len(items))
	var failure error
	for _, item := range items {
		m := arion.ClipboardMeta{}
		for _, path := range item.Payload.Files {
			f := arion.ClipboardFile{Name: filepath.Base(path), Path: path}
			if info, err := os.Stat(path); err == nil {
				f.Available = info.Mode().IsRegular()
				f.Directory = info.IsDir()
			}
			m.Files = append(m.Files, f)
		}
		if item.Payload.Kind == domain.ClipImage {
			m.Thumbnail = cached[item.ID].Thumbnail
			if m.Thumbnail == nil {
				var err error
				m.Thumbnail, err = clipBitmap(item.Payload.PNG, 96)
				if err != nil {
					failure = err
				}
			}
		}
		meta[item.ID] = m
	}
	return meta, failure
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
