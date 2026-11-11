//go:build linux && (amd64 || arm64)

package e2e

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// gtk4Frame is the sheet of #1121, which gives GTK 3's frame the look of
// GTK 4's: a flat header bar and round title buttons of 24 px.
const gtk4Frame = `decoration { border-radius: 12px; box-shadow: 0 3px 9px 1px rgba(0, 0, 0, .28); }
.maximized decoration, .fullscreen decoration, .tiled decoration { border-radius: 0; }
.csd.popup decoration, tooltip.csd decoration { border-radius: 5px; }
headerbar { min-height: 46px; padding: 0 6px; background-color: #f9fafb;
            background-image: none; border-width: 0; box-shadow: none; }
button.titlebutton { min-width: 24px; min-height: 24px; margin: 0 3px; padding: 0;
                     border: none; border-radius: 9999px; background-color: transparent; }`

func titleBarStyle() (s linux.TitleBarStyle) {
	mygo.RunOnMain(func() { s = linux.TestTitleBarStyle() })
	return s
}

func titleButtonSizes(w *mygo.Window) (sizes [][2]int) {
	mygo.RunOnMain(func() { sizes = linux.TestTitleButtonSizes(w.NativeHandle()) })
	return sizes
}

func sameColor(a, b [4]float64) bool {
	for i := range a {
		if math.Abs(a[i]-b[i]) > 1e-6 {
			return false
		}
	}
	return true
}

// Theme.SetGTKCSS styles the widgets GTK draws, above the theme: a header
// bar and its title buttons, the window's own among them. A second sheet
// replaces the first rather than adding to it, one that does not parse
// fails and leaves the sheet before, and "" gives the theme back (#1121).
func TestGTKCSS(t *testing.T) {
	t.Cleanup(func() { _ = mygo.Theme.SetGTKCSS("") })
	theme := titleBarStyle()
	if theme.Button == [2]int{24, 24} {
		t.Fatalf("the theme's title buttons are 24x24 already")
	}
	w := newWindow(t, mygo.WindowOptions{Width: 400, Height: 300, TitleBarStyle: mygo.TitleBarHidden,
		Content: ui.View(func(c *ui.Context) { ui.Text(c, "GTK CSS") })})

	// From a goroutine other than the main one, as tests run.
	if err := mygo.Theme.SetGTKCSS(gtk4Frame); err != nil {
		t.Fatal(err)
	}
	s := titleBarStyle()
	if s.Button != [2]int{24, 24} {
		t.Errorf("title button = %v, want 24x24", s.Button)
	}
	if want := [4]float64{249.0 / 255, 250.0 / 255, 251.0 / 255, 1}; !sameColor(s.Background, want) {
		t.Errorf("header bar background = %v, want #f9fafb", s.Background)
	}
	// The window's title buttons, where the desktop's layout has any.
	if len(titleButtonSizes(w)) > 0 {
		var sizes [][2]int
		if !wait.Poll(func() bool {
			sizes = titleButtonSizes(w)
			return len(sizes) > 0 && !slices.ContainsFunc(sizes, func(s [2]int) bool { return s != [2]int{24, 24} })
		}) {
			t.Errorf("the window's title buttons are %v, never 24x24", sizes)
		}
	}

	// A second sheet replaces the first: the header bar has the theme's
	// background again.
	const big = `button.titlebutton { min-width: 40px; min-height: 40px; padding: 0; border: none; }`
	if err := mygo.Theme.SetGTKCSS(big); err != nil {
		t.Fatal(err)
	}
	s = titleBarStyle()
	if s.Button != [2]int{40, 40} {
		t.Errorf("title button after another sheet = %v, want 40x40", s.Button)
	}
	if !sameColor(s.Background, theme.Background) {
		t.Errorf("header bar background after another sheet = %v, want the theme's %v", s.Background, theme.Background)
	}

	// A sheet that does not parse fails, and the one before stays.
	err := mygo.Theme.SetGTKCSS(`button.titlebutton { min-width: nonsense; }`)
	if err == nil || !strings.HasPrefix(err.Error(), "mygo: Theme.SetGTKCSS: ") {
		t.Errorf("SetGTKCSS of a sheet that does not parse = %v, want GTK's error", err)
	}
	if s := titleBarStyle(); s.Button != [2]int{40, 40} {
		t.Errorf("title button after a sheet that does not parse = %v, want 40x40 still", s.Button)
	}

	// No sheet, the theme's style.
	if err := mygo.Theme.SetGTKCSS(""); err != nil {
		t.Fatal(err)
	}
	if s := titleBarStyle(); s != theme {
		t.Errorf("style without a sheet = %+v, want the theme's %+v", s, theme)
	}
}

// gtkCSSBeforeRun sets a sheet on the main goroutine before Run, which
// keeps it for Run to load, and returns whether GTK styles with it. Main
// thread only.
func gtkCSSBeforeRun() (applied func() bool) {
	err := mygo.Theme.SetGTKCSS(`button.titlebutton { min-width: 32px; min-height: 32px; padding: 0; border: none; }`)
	return func() bool { return err == nil && linux.TestTitleBarStyle().Button == [2]int{32, 32} }
}
