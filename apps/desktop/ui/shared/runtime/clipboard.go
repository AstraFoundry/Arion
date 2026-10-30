package runtime

import (
	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
	"github.com/egoist/mygo/ui"
)

type Clip = domain.Clip
type ClipPayload = domain.ClipPayload
type ClipboardPreferences = application.ClipboardPreferences
type ClipKind = domain.ClipKind
type ClipLimits = domain.ClipLimits

const (
	ClipText  = domain.ClipText
	ClipLink  = domain.ClipLink
	ClipImage = domain.ClipImage
	ClipFiles = domain.ClipFiles
)

func ClipboardText(text string) ClipPayload { return domain.ClipTextPayload(text) }
func DefaultClipboardPreferences() ClipboardPreferences {
	return application.DefaultClipboardPreferences()
}

func FindClips(items []Clip, query string, kind ClipKind) []Clip {
	return domain.FindClips(items, query, kind)
}

type ClipboardFile struct {
	Name, Path           string
	Directory, Available bool
}
type ClipboardMeta struct {
	Files     []ClipboardFile
	Thumbnail *ui.Bitmap
}

type Clipboard struct {
	Items          []Clip
	Meta           map[string]ClipboardMeta
	Busy           bool
	CopyOnly       bool
	Using          bool
	Error          copy.Key
	ShortcutFailed bool
}

// ClipboardShortcutKey resolves legacy accelerator spelling and recorded chords.
func ClipboardShortcutKey(platform, raw string) (HoldKey, bool) {
	return application.ClipboardShortcutKey(platform, raw)
}
