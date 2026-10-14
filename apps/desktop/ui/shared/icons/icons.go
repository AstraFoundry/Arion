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
	ArrowUp           = load("arrow-up")
	AudioLines        = load("audio-lines")
	BookOpen          = load("book-open")
	Briefcase         = load("briefcase")
	Check             = load("check")
	ChevronRight      = load("chevron-right")
	ChevronsUpDown    = load("chevrons-up-down")
	CircleCheck       = load("circle-check")
	Clipboard         = load("clipboard")
	Code              = load("code")
	Copy              = load("copy")
	Cpu               = load("cpu")
	Download          = load("download")
	Globe             = load("globe")
	GripVertical      = load("grip-vertical")
	HardDriveDownload = load("hard-drive-download")
	Info              = load("info")
	Keyboard          = load("keyboard")
	Lock              = load("lock")
	Mail              = load("mail")
	MessageCircle     = load("message-circle")
	Mic               = load("mic")
	MicOff            = load("mic-off")
	Power             = load("power")
	RotateCw          = load("rotate-cw")
	Search            = load("search")
	Settings          = load("settings")
	ShieldCheck       = load("shield-check")
	Sparkles          = load("sparkles")
	TriangleAlert     = load("triangle-alert")
	Type              = load("type")
	X                 = load("x")
	Zap               = load("zap")
)
