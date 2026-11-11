package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// AccessDisabled tells assistive technology that an element and those
// inside it are unavailable (aria-disabled), while they keep the focus,
// the Tab order, the pointer and their look, unlike Disabled: assistive
// technology focuses them, but neither presses nor edits them.
func TestAccessDisabled(t *testing.T) {
	var deleted, sent, closed int
	var note, name string
	locked := true
	var isDisabled bool
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			del := Button(c, "Delete project").AccessDisabled(locked)
			isDisabled = del.IsDisabled()
			if del.Clicked() {
				deleted++
			}
			Row(c).AccessDisabled(locked).Children(func() {
				if Button(c, "Send").Clicked() {
					sent++
				}
				TextInput(c, &note).Label("Note")
			})
			if Button(c, "Close").Disabled(true).Clicked() {
				closed++
			}
			TextInput(c, &name).Label("Name")
		})
	}, 400, 300)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	accessOf := func(t *testing.T, tt *Tester, r Role, name string) AccessInfo {
		t.Helper()
		a, ok := tt.AccessByRole(r, name)
		if !ok {
			t.Fatalf("no element of role %d named %q", r, name)
		}
		return a
	}

	for _, label := range []string{"Delete project", "Send", "Close"} {
		a := accessOf(t, tt, RoleButton, label)
		if !a.Disabled {
			t.Errorf("%s is not disabled", label)
		}
		// Only what Disabled disables leaves the focus.
		if want := label != "Close"; a.Focusable != want {
			t.Errorf("%s focusable %v, want %v", label, a.Focusable, want)
		}
	}
	if isDisabled {
		t.Error("AccessDisabled made the element IsDisabled")
	}
	del := accessNode(t, tt.h.access, platform.RoleButton, "Delete project")
	if del.Actions&platform.ActionFocus == 0 || del.Actions&platform.ActionPress != 0 {
		t.Errorf("Delete project's actions %b, want focus without press", del.Actions)
	}
	field := accessNode(t, tt.h.access, platform.RoleTextField, "Note")
	if field.States&platform.AccessDisabled == 0 || field.Actions&platform.ActionSetValue != 0 || field.States&platform.AccessFocusable == 0 {
		t.Errorf("Note: %+v", field)
	}

	// Tab reaches it, in its place, and the keys and the pointer still
	// press it, for the view to ignore.
	tt.Key(0, KeyTab)
	if tt.h.access.Focus != del.ID {
		t.Errorf("Tab focused %d, want Delete project (%d)", tt.h.access.Focus, del.ID)
	}
	tt.Key(0, KeyEnter)
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Close"); err != nil {
		t.Fatal(err)
	}
	if deleted != 1 || sent != 1 || closed != 0 {
		t.Errorf("presses: deleted %d, sent %d, closed %d; want 1, 1, 0", deleted, sent, closed)
	}

	// Assistive technology focuses it, but neither presses nor edits it.
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: del.ID, Action: platform.AccessPress})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "hi"})
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessInsertText, Text: "hi"})
	if deleted != 1 || note != "" {
		t.Errorf("assistive technology operated what is unavailable: deleted %d, note %q", deleted, note)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessFocus})
	if tt.h.access.Focus != field.ID {
		t.Errorf("assistive technology could not focus Note: focus %d", tt.h.access.Focus)
	}

	// Available again, it is pressed as any other.
	locked = false
	tt.Frame()
	if a := accessOf(t, tt, RoleButton, "Send"); a.Disabled || !a.Focusable {
		t.Errorf("Send once unlocked: %+v", a)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: del.ID, Action: platform.AccessPress})
	if deleted != 2 {
		t.Errorf("pressing Delete project once unlocked: deleted %d", deleted)
	}
}

// AccessDisabled leaves the element's look as it is: it paints as it
// would without it, where Disabled dims it.
func TestAccessDisabledLook(t *testing.T) {
	off := false
	view := func(c *Context) {
		Box(c).Size(100, 100).Background(RGB(0x20, 0x60, 0xd0)).AccessDisabled(off)
	}
	tt := NewTester(view, 100, 100)
	want := tt.Image().RGBAAt(50, 50)
	off = true
	tt.Frame()
	if got := tt.Image().RGBAAt(50, 50); got != want {
		t.Errorf("AccessDisabled painted %v, want %v", got, want)
	}
}
