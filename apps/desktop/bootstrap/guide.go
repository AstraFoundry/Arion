package bootstrap

import (
	"log"
	"time"

	uiapp "arion.local/desktop/ui/app"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

// The consent guide: System Settings opens on a privacy list that macOS
// never prompts for again, and a nonactivating panel under its window
// offers Arion's app to drag into the list. The panel follows the window,
// shows only while System Settings is frontmost, and closes when consent
// arrives, when the user closes it, or when System Settings quits.
const (
	// guideTick paces the window-server reads: smooth enough to follow a
	// dragged window, and only while a guide is open.
	guideTick = time.Second / 30
	// guideLaunch bounds the wait for System Settings to open.
	guideLaunch = 10 * time.Second
	// guideGone is how long System Settings may be missing once seen, as
	// it swaps panes, before the guide closes.
	guideGone = 400 * time.Millisecond
	// guideSidebar is System Settings' sidebar width; the guide lines up
	// with the pane beside it.
	guideSidebar  = 230
	guideMinWidth = 360
	guideGap      = 8
	guideInset    = 8
)

type guideSession struct {
	pane     arion.GuidePane
	window   *mygo.Window
	returnTo *mygo.Window
	stop     chan struct{}
}

// guideFrame is one observation of the settings window by the worker.
type guideFrame struct {
	settings SettingsWindow
	granted  bool
	ended    bool
}

// guideConsent reads the consent a pane grants.
func (a *App) guideConsent(pane arion.GuidePane) arion.Consent {
	if pane == arion.GuideInputMonitoring {
		return a.platform.InputMonitoring()
	}
	return a.platform.Accessibility()
}

// openGuide opens System Settings on pane with the drag guide under it. It
// reports false where no guide can exist: outside an application bundle
// there is nothing to drag, and the guide panel hangs from the tray icon.
func (a *App) openGuide(pane arion.GuidePane) bool {
	app := a.platform.AppBundle()
	if app == "" || a.tray == nil {
		return false
	}
	drag, err := transfer.FileData(app)
	if err != nil {
		log.Printf("consent guide unavailable: %v", err)
		return false
	}
	if a.guide != nil && a.guide.pane != pane {
		a.closeGuide(false)
	}
	a.platform.OpenGuidePane(pane)
	if a.guide != nil {
		return true
	}
	view := uiapp.Guide(a.ui)
	window := mygo.NewFlyout(mygo.FlyoutOptions{Tray: a.tray, Width: guideMinWidth, Height: uiapp.GuideHeight,
		Shadow: true, Hidden: true, Content: view.Content})
	g := &guideSession{pane: pane, window: window, returnTo: mygo.FocusedWindow(), stop: make(chan struct{})}
	a.guide = g
	a.state.Guide = arion.Guide{Pane: pane, App: drag}
	a.track(window, view, func() { a.endGuide(g) })
	go a.followGuide(g)
	return true
}

// closeGuide closes the guide; back returns the user to the Arion window
// that asked for consent.
func (a *App) closeGuide(back bool) {
	g := a.guide
	if g == nil {
		return
	}
	g.window.Close()
	a.endGuide(g)
	if _, open := a.views[g.returnTo]; back && open {
		mygo.App.Focus()
		g.returnTo.Focus()
	}
}

func (a *App) endGuide(g *guideSession) {
	if a.guide != g {
		return
	}
	a.guide = nil
	a.state.Guide = arion.Guide{}
	close(g.stop)
}

// followGuide observes the settings window off the main thread and posts
// only changes.
func (a *App) followGuide(g *guideSession) {
	ticker := time.NewTicker(guideTick)
	defer ticker.Stop()
	started := time.Now()
	var seen, last time.Time
	var previous guideFrame
	for {
		select {
		case <-g.stop:
			return
		case now := <-ticker.C:
			frame := guideFrame{settings: a.platform.SettingsWindow(), granted: a.guideConsent(g.pane) == arion.ConsentGranted}
			if frame.settings.Running {
				seen, last = now, now
			}
			frame.ended = (seen.IsZero() && now.Sub(started) > guideLaunch) || (!seen.IsZero() && now.Sub(last) > guideGone)
			if frame == previous {
				continue
			}
			previous = frame
			a.post(func() { a.updateGuide(g, frame) })
		}
	}
}

func (a *App) updateGuide(g *guideSession, frame guideFrame) {
	if a.guide != g {
		return
	}
	switch {
	case frame.granted:
		a.refreshPermissions()
		a.updateTray()
		a.closeGuide(true)
		return
	case frame.ended:
		a.closeGuide(false)
		return
	}
	settings := frame.settings
	if !settings.Visible || !settings.Frontmost {
		g.window.Hide()
		return
	}
	area := mygo.Screen.DisplayMatching(settings.Bounds).WorkArea
	g.window.SetBounds(guideBounds(settings.Bounds, area, uiapp.GuideHeight))
	if !g.window.IsVisible() {
		a.state.Guide.Shown++
		g.window.ShowInactive()
	}
}

// guideBounds places the guide under the settings window, aligned with the
// pane beside its sidebar, inside the display's work area. Where the
// window reaches the bottom of the display, the guide overlaps its foot.
func guideBounds(settings, area mygo.Rectangle, height int) mygo.Rectangle {
	width := min(max(settings.Width-guideSidebar, guideMinWidth), area.Width-2*guideInset)
	x := clamp(settings.X+guideSidebar, area.X+guideInset, area.X+area.Width-guideInset-width)
	y := clamp(settings.Y+settings.Height+guideGap, area.Y+guideInset, area.Y+area.Height-guideInset-height)
	return mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
