package application

import (
	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// UsageMeter folds a session's events into the usage it adds to the day's
// counts. Speech time runs from capture start to stop; only text that was
// committed counts, so cancelled, silent and failed sessions add nothing.
// Use it from one goroutine, in event order.
type UsageMeter struct {
	session   string
	capturing float64
	usage     domain.Usage
}

// Observe returns the session's usage once its commit completes.
func (m *UsageMeter) Observe(e contracts.Event) (domain.Usage, bool) {
	if e.Kind == contracts.SessionStateChanged && e.State == domain.Preparing {
		*m = UsageMeter{session: e.SessionID}
		return domain.Usage{}, false
	}
	if e.SessionID != m.session {
		return domain.Usage{}, false
	}
	switch e.Kind {
	case contracts.SessionStateChanged:
		switch e.State {
		case domain.Capturing:
			m.capturing = e.AtMs
		case domain.Finalizing:
			m.usage.SpeechMs = int64(e.AtMs - m.capturing)
		}
	case contracts.CorrectionReady:
		m.usage.Scene = ""
		if e.Rewrite == string(domain.RewriteAccepted) && e.Source != e.Text {
			m.usage.Scene = domain.Scene(e.Scene)
		}
	case contracts.CommitResultReady:
		if e.State != domain.Completed {
			return domain.Usage{}, false
		}
		usage := m.usage
		usage.Words = domain.CountWords(e.Text)
		m.session = ""
		return usage, true
	}
	return domain.Usage{}, false
}
