package bootstrap

import (
	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/urlopen"
)

func (a *App) syncSelectionWatch(before, next bool) {
	if a.platform == nil || before == next {
		return
	}
	if !next {
		a.platform.StopSelectionWatch()
		a.publishSelection(application.SelectionOffer{}, false)
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

func (a *App) selectionLoop() {
	for range a.selectionClicks {
		a.applySelectionRead()
	}
}

func (a *App) applySelectionRead() {
	offer, ok, err := application.OfferAfterMouseUp(a.preferences().Selection, a.readSelection)
	if err != nil {
		if domain.CodeOf(err) == domain.CodePermissionDenied {
			a.record("selection_watch", err)
		}
		a.publishSelection(application.SelectionOffer{}, false)
		return
	}
	a.publishSelection(offer, ok)
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

func (a *App) publishSelection(offer application.SelectionOffer, ok bool) {
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
			return
		}
		a.state.Selection = application.SelectionOffer{}
	}
	if a.queueReady == nil {
		apply()
		return
	}
	a.post(apply)
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

func (x actions) OpenSelection(actionID, question string) error {
	offer, ok := x.a.currentSelection()
	if !ok {
		return application.ErrSelectionEmpty
	}
	if err := x.a.confirmSelectionReadable(); err != nil {
		return err
	}
	raw, err := x.a.preferences().Selection.SelectionURL(actionID, offer.Text, question)
	if err != nil {
		return err
	}
	return x.a.selectionOpener()(raw)
}

func (a *App) selectionOpener() func(string) error {
	if a.openSelectionURL != nil {
		return a.openSelectionURL
	}
	return urlopen.Open
}
