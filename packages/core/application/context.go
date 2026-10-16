package application

import (
	"sync"
	"time"

	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// ContextPolicy says which context a session's rewrite may read (009,
// ADR 013). Every source stays on the device.
type ContextPolicy struct {
	// Cursor is the field's text around the cursor.
	Cursor bool
	// Screen is the text recognized in the focused window.
	Screen bool
	// History is the user's recent dictations in the same place.
	History bool
}

// Any reports whether some source is allowed.
func (p ContextPolicy) Any() bool { return p.Cursor || p.Screen || p.History }

// contextWait bounds how long the final rewrite waits for context that is
// still being read; previews never wait.
const contextWait = 1500 * time.Millisecond

// History keeps the user's recent dictations per place (domain.SceneKey) in
// memory only. Items expire, and Clear forgets everything.
type History struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	items map[string][]historyItem
}

type historyItem struct {
	text string
	at   time.Time
}

// NewHistory keeps items for ttl.
func NewHistory(ttl time.Duration) *History {
	return &History{ttl: ttl, now: time.Now, items: map[string][]historyItem{}}
}

// Add records a dictation inserted in place.
func (h *History) Add(place, text string) {
	if h == nil || place == "" || text == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	items := append(h.items[place], historyItem{text: text, at: h.now()})
	if len(items) > domain.MaxHistoryItems {
		items = items[len(items)-domain.MaxHistoryItems:]
	}
	h.items[place] = items
}

// Recent returns place's unexpired dictations, oldest first.
func (h *History) Recent(place string) []string {
	if h == nil || place == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	kept := h.items[place][:0]
	for _, item := range h.items[place] {
		if h.now().Sub(item.at) <= h.ttl {
			kept = append(kept, item)
			out = append(out, item.text)
		}
	}
	if len(kept) == 0 {
		delete(h.items, place)
	} else {
		h.items[place] = kept
	}
	return out
}

// Clear forgets every place.
func (h *History) Clear() {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.items = map[string][]historyItem{}
	h.mu.Unlock()
}

// startContext begins reading the session's context after its target was
// captured. Secure fields never get here (DSK-08); denied apps read nothing.
func (c *Coordinator) startContext(r *run) {
	l := r.live
	if l == nil || !l.policy.Enabled || !l.policy.Context.Any() || !r.hasTarget || domain.ContextDenied(r.target.Application) {
		return
	}
	r.place = domain.SceneKey(r.target.SceneHints())
	r.contextReady = make(chan struct{})
	policy := l.policy.Context
	go func() {
		defer close(r.contextReady)
		var rc domain.RewriteContext
		if policy.History {
			rc.History = c.deps.History.Recent(r.place)
		}
		if _, input := c.targets(r); input != nil && (policy.Cursor || policy.Screen) {
			started := time.Now()
			snap, err := input.ReadContext(r.ctx, r.identity, r.target.Target)
			c.diagnostic(r, ports.DiagnosticContext, started, err)
			if err == nil && snap.SafeToRead {
				if policy.Cursor {
					rc.Before, rc.After = snap.Before, snap.After
				}
				if policy.Screen {
					rc.Screen = snap.Screen
				}
			}
		}
		r.rewriteContext = rc.Bounded()
	}()
}

// contextFor returns the session's context once read, waiting at most wait.
func (r *run) contextFor(wait time.Duration) domain.RewriteContext {
	if r.contextReady == nil {
		return domain.RewriteContext{}
	}
	if wait <= 0 {
		select {
		case <-r.contextReady:
			return r.rewriteContext
		default:
			return domain.RewriteContext{}
		}
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-r.contextReady:
		return r.rewriteContext
	case <-timer.C:
		return domain.RewriteContext{}
	case <-r.ctx.Done():
		return domain.RewriteContext{}
	}
}

// remember adds an inserted dictation to its place's history.
func (c *Coordinator) remember(r *run, text string) {
	if r.live != nil && r.live.policy.Context.History {
		c.deps.History.Add(r.place, text)
	}
}
