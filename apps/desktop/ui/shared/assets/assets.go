// Package assets embeds the bitmaps the native UI draws: the app mark and
// the photograph illustrations sit on. See README.md for sources.
package assets

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
)

var (
	//go:embed mark.png
	markPNG []byte
	//go:embed backdrop.jpg
	backdropJPEG []byte
	//go:embed astrafoundry.png
	teamPNG []byte
	//go:embed feishu.png
	feishuPNG []byte
)

// Mark is the app icon, drawn wherever Arion names itself.
var Mark = mustDecode(markPNG)

// Backdrop is the motion-blurred photograph under illustrated mock UI.
var Backdrop = mustDecode(backdropJPEG)

// Team is the AstraFoundry organization avatar, as GitHub shows it.
var Team = mustDecode(teamPNG)

// Feishu is Feishu's logo, which links to the community group.
var Feishu = mustDecode(feishuPNG)

// mustDecode decodes an embedded image; one that does not decode is a
// build defect, not a runtime condition.
func mustDecode(data []byte) *ui.Bitmap {
	bitmap, err := ui.DecodeBitmap(data)
	if err != nil {
		panic("assets: " + err.Error())
	}
	return bitmap
}
