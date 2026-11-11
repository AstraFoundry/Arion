//go:build !js

package mygo

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestDialogFiltersByType passes a filter's MIME types, wildcards too, to
// the system's picker with its extensions.
func TestDialogFiltersByType(t *testing.T) {
	var got []platform.FileFilter
	onMain(func() {
		fb.OnOpenDialog = func(o *platform.OpenDialogOptions) []string {
			got = o.Filters
			return nil
		}
	})
	defer onMain(func() { fb.OnOpenDialog = nil })
	Dialog.Open(OpenDialogOptions{Filters: []FileFilter{
		{Name: "Images", MIMETypes: []string{"image/*"}},
		{Name: "Documents", Extensions: []string{"md"}, MIMETypes: []string{"application/pdf"}},
	}})
	if len(got) != 2 || got[0].Name != "Images" || !slices.Equal(got[0].MIMETypes, []string{"image/*"}) ||
		!slices.Equal(got[1].Extensions, []string{"md"}) || !slices.Equal(got[1].MIMETypes, []string{"application/pdf"}) {
		t.Errorf("the picker gets %+v", got)
	}
}
