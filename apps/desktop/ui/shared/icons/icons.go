// Package icons exposes the vendored Lucide icon set (ISC, see svg/LICENSE),
// the Google and GitHub marks of the AstraFoundry sign-in methods, kept in
// their brand colors, and the cloud polishing providers' marks from LobeHub
// Icons (MIT, see svg/LICENSE-lobehub), and Discord's mark from SVG Logos
// (CC0, see svg/LICENSE-svg-logos). Monochrome marks are tinted with
// ui.Icon; Gemini, DeepSeek and Discord keep their colors and are drawn
// with ui.Image.
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
	ArrowDown         = load("arrow-down")
	ArrowRight        = load("arrow-right")
	ArrowUp           = load("arrow-up")
	AudioLines        = load("audio-lines")
	BookOpen          = load("book-open")
	Briefcase         = load("briefcase")
	Check             = load("check")
	ChartLine         = load("chart-line")
	ChevronLeft       = load("chevron-left")
	ChevronRight      = load("chevron-right")
	ExternalLink      = load("external-link")
	ChevronsUpDown    = load("chevrons-up-down")
	CircleCheck       = load("circle-check")
	CircleUser        = load("circle-user")
	Clipboard         = load("clipboard")
	Clock             = load("clock")
	Code              = load("code")
	Copy              = load("copy")
	Cpu               = load("cpu")
	Download          = load("download")
	Gauge             = load("gauge")
	GitHub            = load("github")
	Globe             = load("globe")
	Google            = load("google")
	GripVertical      = load("grip-vertical")
	HardDriveDownload = load("hard-drive-download")
	Info              = load("info")
	Keyboard          = load("keyboard")
	Lock              = load("lock")
	Mail              = load("mail")
	MessageCircle     = load("message-circle")
	Mic               = load("mic")
	MicOff            = load("mic-off")
	Pencil            = load("pencil")
	Plus              = load("plus")
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

// Cloud polishing provider marks. ChatGPT uses the OpenAI mark.
var (
	OpenAI     = load("lobe-openai")
	Anthropic  = load("lobe-anthropic")
	OpenRouter = load("lobe-openrouter")
	Gemini     = load("lobe-gemini")
	// Discord is the community server's mark, in its brand color.
	Discord  = load("logos-discord")
	DeepSeek = load("lobe-deepseek")
)
