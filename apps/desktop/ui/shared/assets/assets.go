// Package assets embeds the bitmaps the native UI draws (the app mark and
// the photograph illustrations sit on), and the text and typeface the
// onboarding typewriter types. See README.md for sources.
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

// Arion is Herodotus's account of Arion and the dolphin (Histories 1.23–24)
// in the original Greek, one paragraph per line: the typewriter figure
// types it.
//
//go:embed arion.txt
var Arion string

// Serif is Gentium Book Plus Regular, a serif with polytonic Greek, whose
// outlines the typewriter figure draws.
//
//go:embed GentiumBookPlus-Regular.ttf
var Serif []byte

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
