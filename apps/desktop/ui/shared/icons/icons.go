// Package icons exposes the vendored Lucide icon set (ISC, see svg/LICENSE).
package icons

import (
	"embed"

	"github.com/egoist/mygo/ui"
)

//go:embed svg/*.svg
var files embed.FS

func load(name string) *ui.SVG {
	data, err := files.ReadFile("svg/" + name + ".svg")
	if err != nil {
		panic(err)
	}
	return ui.MustParseSVG(data)
}

var (
	Accessibility     = load("accessibility")
	ArrowRight        = load("arrow-right")
	AudioLines        = load("audio-lines")
	BookOpen          = load("book-open")
	Check             = load("check")
	ChevronRight      = load("chevron-right")
	CircleCheck       = load("circle-check")
	Clipboard         = load("clipboard")
	Copy              = load("copy")
	Cpu               = load("cpu")
	Download          = load("download")
	Globe             = load("globe")
	HardDriveDownload = load("hard-drive-download")
	Info              = load("info")
	Keyboard          = load("keyboard")
	Lock              = load("lock")
	Mic               = load("mic")
	MicOff            = load("mic-off")
	Power             = load("power")
	RotateCw          = load("rotate-cw")
	Settings          = load("settings")
	ShieldCheck       = load("shield-check")
	Sparkles          = load("sparkles")
	TriangleAlert     = load("triangle-alert")
	Type              = load("type")
	X                 = load("x")
	Zap               = load("zap")
)
