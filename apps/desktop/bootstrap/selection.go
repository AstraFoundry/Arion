package bootstrap

import (
	"strings"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/urlopen"
	"github.com/egoist/mygo"
)

func (a *App) syncSelectionWatch(before, next bool) {
	if a.platform == nil || before == next {
		return
	}
	if !next {
		a.platform.StopSelectionWatch()
		a.publishSelection(application.SelectionOffer{}, false, mygo.Rectangle{}, false, false)
		return
	}
	a.ensureSelectionLoop()
	if err := a.platform.StartSelectionWatch(a.signalSelection); err != nil {
		a.record("selection_watch", err)
	}
}

func (a *App) stopSelectionWatch() {
	if a.platform != nil {
		a.platform.StopSelectionWatch()
	}
}

func (a *App) ensureSelectionLoop() {
	a.selectionStarted.Do(func() {
		a.selectionClicks = make(chan struct{}, 1)
		go a.selectionLoop()
	})
}

func (a *App) signalSelection() {
	if a.selectionClicks == nil {
		return
	}
	select {
	case a.selectionClicks <- struct{}{}:
	default:
	}
}

// selectionLoop skips a release over the bar or a popup beside it: that
// click chooses in the bar, and the question field taking the keyboard is
// not a new selection.
func (a *App) selectionLoop() {
	for range a.selectionClicks {
		if a.pointerOnSelection() {
			continue
		}
		a.applySelectionRead()
	}
}

func (a *App) pointerOnSelection() bool {
	inside := false
	a.postWait(func() {
		windows := a.selectionWindows()
		if len(windows) == 0 || !a.selectionWin.IsVisible() {
			return
		}
		cursor := mygo.Screen.CursorScreenPoint()
		for _, window := range windows {
			if window.IsVisible() && selectionContains(window.Bounds(), cursor) {
				inside = true
				return
			}
		}
	})
	return inside
}

func selectionContains(bounds mygo.Rectangle, p mygo.Point) bool {
	return p.X >= bounds.X && p.Y >= bounds.Y && p.X < bounds.X+bounds.Width && p.Y < bounds.Y+bounds.Height
}

func (a *App) applySelectionRead() {
	offer, ok, err := application.OfferAfterMouseUp(a.preferences().Selection, time.Now(), a.readSourcedSelection)
	if err != nil {
		if domain.CodeOf(err) == domain.CodePermissionDenied {
			a.record("selection_watch", err)
		}
		a.publishSelection(application.SelectionOffer{}, false, mygo.Rectangle{}, false, true)
		return
	}
	var anchor mygo.Rectangle
	hasAnchor := false
	if ok && a.platform != nil {
		if x, y, width, height, anchored := a.platform.SelectionAnchor(); anchored {
			anchor = mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
			hasAnchor = true
		}
	}
	a.publishSelection(offer, ok, anchor, hasAnchor, true)
}

func (a *App) readSelection() (application.SelectionReading, error) {
	if a.platform == nil {
		return application.SelectionReading{}, domain.Fail(domain.CodeTargetUnsupported, "selection unavailable")
	}
	text, secure, err := a.platform.ReadSelection()
	if err != nil {
		return application.SelectionReading{}, err
	}
	return application.SelectionReading{Text: text, Secure: secure}, nil
}

// readSourcedSelection is readSelection with the app and site of a
// selection the bar may be offered for.
func (a *App) readSourcedSelection() (application.SelectionReading, error) {
	reading, err := a.readSelection()
	if err == nil && !reading.Secure && strings.TrimSpace(reading.Text) != "" {
		reading.Source = a.platform.SelectionSource()
	}
	return reading, err
}

func (a *App) publishSelection(offer application.SelectionOffer, ok bool, anchor mygo.Rectangle, hasAnchor bool, wait bool) {
	a.selectionMu.Lock()
	a.selectionOK = ok
	if ok {
		a.selectionOffer = offer
	} else {
		a.selectionOffer = application.SelectionOffer{}
	}
	a.selectionMu.Unlock()
	apply := func() {
		if ok {
			a.state.Selection = offer
			a.presentSelection(anchor, hasAnchor)
			return
		}
		a.state.Selection = application.SelectionOffer{}
		a.dismissSelection()
	}
	if a.queueReady != nil && !wait {
		a.post(apply)
		return
	}
	a.postWait(apply)
}

// postWait runs fn on the UI queue and returns after it ran.
func (a *App) postWait(fn func()) {
	if a.queueReady == nil {
		fn()
		return
	}
	done := make(chan struct{})
	a.post(func() {
		fn()
		close(done)
	})
	<-done
}

func (a *App) currentSelection() (application.SelectionOffer, bool) {
	a.selectionMu.Lock()
	defer a.selectionMu.Unlock()
	if !a.selectionOK {
		return application.SelectionOffer{}, false
	}
	return a.selectionOffer, true
}

func (a *App) confirmSelectionReadable() error {
	if !a.preferences().Selection.Enabled {
		return application.ErrSelectionAction
	}
	reading, err := a.readSelection()
	if err != nil {
		return err
	}
	if reading.Secure {
		return domain.Fail(domain.CodeSecureFieldBlocked, "secure field")
	}
	return nil
}

func (a *App) performSelection(apply func(string) error) error {
	offer, ok := a.currentSelection()
	if !ok {
		return application.ErrSelectionEmpty
	}
	if err := a.confirmSelectionReadable(); err != nil {
		return err
	}
	return apply(offer.Text)
}

func (x actions) CopySelection() error {
	return x.a.performSelection(func(text string) error {
		if !x.a.copyTranscript(text) {
			return domain.Fail(domain.CodeClipboardUnavailable, "clipboard")
		}
		return nil
	})
}

func (x actions) AddSelectionVocabulary() error {
	return x.a.performSelection(func(text string) error {
		return x.AddHotword(text)
	})
}

func (x actions) OpenSelection(actionID, clientID, question string) error {
	offer, ok := x.a.currentSelection()
	if !ok {
		return application.ErrSelectionEmpty
	}
	if err := x.a.confirmSelectionReadable(); err != nil {
		return err
	}
	raw, err := x.a.preferences().Selection.SelectionURL(actionID, clientID, offer.Text, question)
	if err != nil {
		return err
	}
	return x.a.selectionOpener()(raw)
}

// DisableSelection saves the scope for the current offer's source and
// dismisses the bar; turning the master switch off stops the watch too.
func (x actions) DisableSelection(scope string) error {
	offer, ok := x.a.currentSelection()
	if !ok {
		return application.ErrSelectionEmpty
	}
	now := time.Now()
	probe := x.a.preferences().Selection
	if err := probe.Disable(scope, offer.Source, now); err != nil {
		return err
	}
	x.UpdatePreferences(func(p *application.Preferences) { _ = p.Selection.Disable(scope, offer.Source, now) })
	saved := x.a.preferences().Selection
	if saved.Enabled && !saved.Paused(now) && !saved.DisabledFor(offer.Source) {
		return application.ErrSelectionAction
	}
	x.a.publishSelection(application.SelectionOffer{}, false, mygo.Rectangle{}, false, false)
	return nil
}

func (a *App) selectionOpener() func(string) error {
	if a.openSelectionURL != nil {
		return a.openSelectionURL
	}
	return urlopen.Open
}
