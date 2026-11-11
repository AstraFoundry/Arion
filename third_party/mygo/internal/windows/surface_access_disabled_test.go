//go:build windows && (amd64 || arm64)

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// sentEvents records the events a surface sends its window's handler.
type sentEvents struct {
	platform.WindowHandler
	events []platform.SurfaceEvent
}

func (s *sentEvents) SurfaceEvent(ev platform.SurfaceEvent) bool {
	s.events = append(s.events, ev)
	return true
}

// An element only unavailable (aria-disabled) is not enabled but keeps
// the keyboard focus, which assistive technology moves to it, as
// Chromium's, while it presses neither it nor one disabled.
func TestUIAUnavailable(t *testing.T) {
	tree := newTestTree(t, false)
	sent := &sentEvents{}
	tree.s = &surface{w: &window{h: sent}}
	tree.update(&platform.AccessTree{Nodes: []platform.AccessNode{
		{ID: 1, Parent: -1, Role: platform.RoleGroup, Label: "Form"},
		{ID: 2, Parent: 0, Role: platform.RoleButton, Label: "Delete project",
			States: platform.AccessDisabled | platform.AccessFocusable, Actions: platform.ActionFocus},
		{ID: 3, Parent: 0, Role: platform.RoleButton, Label: "Close", States: platform.AccessDisabled},
	}})
	del, closeButton := tree.nodes[2], tree.nodes[3]
	for _, c := range []struct {
		e                  *uiaElement
		enabled, focusable bool
	}{{del, false, true}, {closeButton, false, false}} {
		var enabled, focusable variant
		c.e.property(uiaIsEnabledProperty, &enabled)
		c.e.property(uiaIsFocusableProperty, &focusable)
		if enabled != boolVariant(c.enabled) || focusable != boolVariant(c.focusable) {
			t.Errorf("%s: IsEnabled %+v, IsKeyboardFocusable %+v", c.e.n.Label, enabled, focusable)
		}
	}
	if hr := del.act(platform.AccessPress, ""); hr != uiaElementNotEnabled {
		t.Errorf("pressing Delete project: %#x", hr)
	}
	if hr := closeButton.act(platform.AccessFocus, ""); hr != uiaElementNotEnabled {
		t.Errorf("focusing Close: %#x", hr)
	}
	if hr := del.act(platform.AccessFocus, ""); hr != sOK {
		t.Errorf("focusing Delete project: %#x", hr)
	}
	if len(sent.events) != 1 || sent.events[0].Action != platform.AccessFocus || sent.events[0].ID != 2 {
		t.Errorf("events sent: %+v", sent.events)
	}
}
