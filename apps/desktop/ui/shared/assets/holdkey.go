package assets

import (
	"embed"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The dictation key figure, drawn by scripts/build-holdkey-figures.mjs:
// one keyboard per platform and appearance, and over it the chosen key,
// lit, at rest and pressed.
//
//go:embed holdkey/*.svg
var holdKeyFiles embed.FS

var (
	holdKeyMu      sync.Mutex
	holdKeyFigures = map[string]*ui.SVG{}
)

// HoldKeyFigure returns a layer of the figure by name: a platform's
// keyboard ("darwin-dark") or a chosen key ("right_option-light",
// "fn-dark-pressed"). Layers are parsed when first shown, since a platform
// shows only its own.
func HoldKeyFigure(name string) *ui.SVG {
	holdKeyMu.Lock()
	defer holdKeyMu.Unlock()
	if svg, ok := holdKeyFigures[name]; ok {
		return svg
	}
	data, err := holdKeyFiles.ReadFile("holdkey/" + name + ".svg")
	if err != nil {
		panic("assets: " + err.Error())
	}
	svg := ui.MustParseSVG(data)
	holdKeyFigures[name] = svg
	return svg
}
