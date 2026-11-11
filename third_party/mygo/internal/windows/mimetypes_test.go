//go:build windows && (amd64 || arm64)

package windows

import (
	"slices"
	"testing"
)

// TestMIMETypeExtensions finds the extensions of a type, of a kind by
// type or PerceivedType, and of every file.
func TestMIMETypeExtensions(t *testing.T) {
	types := &mimeTypes{exts: []mimeExtension{
		{ext: ".png", contentType: "image/png", perceived: "image"},
		{ext: ".jpg", contentType: "image/jpeg", perceived: "image"},
		{ext: ".jpeg", contentType: "image/jpeg"},
		{ext: ".heic", perceived: "image"},
		{ext: ".mp3", contentType: "audio/mpeg", perceived: "audio"},
	}}
	for _, tc := range []struct {
		mime string
		want []string
	}{
		{"image/jpeg", []string{".jpg", ".jpeg"}},
		{"IMAGE/*", []string{".png", ".jpg", ".jpeg", ".heic"}},
		{"audio/*", []string{".mp3"}},
		{"video/*", nil},
		{"*/*", []string{".*"}},
	} {
		if got := types.extensions(tc.mime); !slices.Equal(got, tc.want) {
			t.Errorf("extensions(%q) = %q, want %q", tc.mime, got, tc.want)
		}
	}
	// The registry has the usual types.
	if got := readMIMETypes().extensions("image/png"); !slices.Contains(got, ".png") {
		t.Errorf("the registry gives image/png %q", got)
	}
}
