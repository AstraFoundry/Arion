//go:build !js

package mygo

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/internal/fake"
	"github.com/egoist/mygo/internal/platform"
)

// webViewMenu is a context menu of the web view's: Back, Reload, a
// separator, and a Copy item, all numbered, with a submenu of Speech.
func webViewMenu() []*platform.MenuItem {
	return []*platform.MenuItem{
		{Native: 1, Name: ContextMenuBack, Label: "Back", Enabled: true, Visible: true},
		{Native: 2, Name: ContextMenuReload, Label: "Reload", Enabled: true, Visible: true},
		{Native: 3, Type: platform.MenuItemSeparator, Enabled: true, Visible: true},
		{Native: 4, Name: ContextMenuCopy, Label: "Copy", Accelerator: "Ctrl+C", Enabled: true, Visible: true},
		{Native: 5, Name: "speech", Label: "Speech", Type: platform.MenuItemSubmenu, Enabled: true, Visible: true, Submenu: &platform.Menu{Items: []*platform.MenuItem{
			{Native: 6, Name: "startSpeaking", Label: "Start Speaking", Enabled: true, Visible: true},
			{Native: 7, Name: "stopSpeaking", Label: "Stop Speaking", Visible: true},
		}}},
	}
}

func askContextMenu(fw *fake.Window, m *platform.ContextMenu) platform.ContextMenuResponse {
	return onMainValue(func() platform.ContextMenuResponse { return fw.H.ContextMenu(m) })
}

// natives lists the numbers of a menu's items, 0 for the app's, -1 for
// those hidden, with their submenus' after them.
func natives(items []*platform.MenuItem) []int {
	var out []int
	for _, it := range items {
		n := it.Native
		if !it.Visible {
			n = -1
		}
		out = append(out, n)
		if it.Submenu != nil {
			out = append(out, natives(it.Submenu.Items)...)
		}
	}
	return out
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Without a listener the web view shows its menu, and the backend need not
// describe it.
func TestContextMenuDefault(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	handles := func() bool { return onMainValue(fw.H.HandlesContextMenu) }
	if handles() {
		t.Error("HandlesContextMenu without a listener")
	}
	if r := askContextMenu(fw, &platform.ContextMenu{Items: webViewMenu()}); !r.Default {
		t.Errorf("answer without a listener = %+v", r)
	}
	off := w.Page().OnContextMenu(func(*ContextMenuEvent) {})
	if !handles() {
		t.Error("no HandlesContextMenu with a listener")
	}
	// A listener that changes nothing leaves the menu to the web view.
	if r := askContextMenu(fw, &platform.ContextMenu{Items: webViewMenu()}); !r.Default {
		t.Errorf("answer of a listener changing nothing = %+v", r)
	}
	off()
	if handles() {
		t.Error("HandlesContextMenu after off")
	}
}

// A listener hears what was clicked and the web view's items with their
// IDs, and the menu it makes of them is what shows.
func TestContextMenuChanged(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	var got *ContextMenuEvent
	var ids []string
	clicked := make(chan *Window, 1)
	w.Page().OnContextMenu(func(e *ContextMenuEvent) {
		got = e
		ids = nil
		for _, it := range e.Menu.Items() {
			ids = append(ids, it.ID)
		}
		e.Menu.Remove(ContextMenuReload, "stopSpeaking")
		e.Menu.ItemByID(ContextMenuCopy).SetLabel("Copy Text")
		e.Menu.ItemByID(ContextMenuBack).SetEnabled(false)
		e.Menu.Append(Separator(), &MenuItem{Label: "Open in Browser", Click: func(_ *MenuItem, w *Window) { clicked <- w }})
	})
	m := &platform.ContextMenu{
		Target: platform.ContextMenuTarget{LinkURL: "https://example.com/", Editable: true},
		Items:  webViewMenu(),
	}
	r := askContextMenu(fw, m)
	if got.LinkURL != "https://example.com/" || !got.Editable || got.SelectionText != "" {
		t.Errorf("event = %+v", got)
	}
	if want := []string{ContextMenuBack, ContextMenuReload, "", ContextMenuCopy, "speech"}; len(ids) != len(want) || ids[0] != want[0] || ids[1] != want[1] || ids[3] != want[3] || ids[4] != want[4] {
		t.Errorf("IDs = %q, want %q", ids, want)
	}
	if r.Default || r.Menu == nil {
		t.Fatalf("answer = %+v, want the changed menu", r)
	}
	items := r.Menu.Items
	if n := natives(items); !sameInts(n, []int{1, 3, 4, 5, 6, 0, 0}) {
		t.Fatalf("items shown = %v", n)
	}
	if items[0].Enabled || items[2].Label != "Copy Text" || items[2].Accelerator != "Ctrl+C" || items[2].Name != ContextMenuCopy {
		t.Errorf("WebKit's items as changed = %+v, %+v", items[0], items[2])
	}
	app := items[len(items)-1]
	if app.Label != "Open in Browser" {
		t.Errorf("the app's item = %+v", app)
	}
	onMain(func() { fw.H.MenuItemClicked(app.ID) })
	select {
	case cw := <-clicked:
		if cw != w {
			t.Errorf("Click got %v, want the window", cw)
		}
	default:
		t.Error("the app's item chosen ran no Click")
	}
}

// A listener's Click on a web view's item runs in its place: the item is
// the app's then.
func TestContextMenuNativeClick(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	copied := 0
	w.Page().OnContextMenu(func(e *ContextMenuEvent) {
		e.Menu.ItemByID(ContextMenuCopy).Click = func(*MenuItem, *Window) { copied++ }
	})
	r := askContextMenu(fw, &platform.ContextMenu{Items: webViewMenu()})
	if r.Menu == nil {
		t.Fatalf("answer = %+v, want the changed menu", r)
	}
	it := r.Menu.Items[3]
	if it.Native != 0 || it.Name != "" || it.Label != "Copy" {
		t.Fatalf("Copy with a Click = %+v, want the app's", it)
	}
	onMain(func() { fw.H.MenuItemClicked(it.ID) })
	if copied != 1 {
		t.Errorf("Click ran %d times", copied)
	}
}

// PreventDefault, and a nil Menu, show no menu; a menu of the app's shows
// in place of the web view's; a web view's item shows once.
func TestContextMenuReplaced(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	var listen func(e *ContextMenuEvent)
	w.Page().OnContextMenu(func(e *ContextMenuEvent) { listen(e) })
	ask := func() platform.ContextMenuResponse {
		return askContextMenu(fw, &platform.ContextMenu{Items: webViewMenu()})
	}
	listen = func(e *ContextMenuEvent) { e.PreventDefault() }
	if r := ask(); r.Default || r.Menu != nil {
		t.Errorf("PreventDefault: answer = %+v", r)
	}
	listen = func(e *ContextMenuEvent) { e.Menu = nil }
	if r := ask(); r.Default || r.Menu != nil {
		t.Errorf("nil Menu: answer = %+v", r)
	}
	listen = func(e *ContextMenuEvent) { e.Menu = NewMenu([]*MenuItem{{Label: "Mine"}}) }
	if r := ask(); r.Menu == nil || len(r.Menu.Items) != 1 || r.Menu.Items[0].Label != "Mine" || r.Menu.Items[0].Native != 0 {
		t.Errorf("a menu of the app's: answer = %+v", r)
	}
	listen = func(e *ContextMenuEvent) {
		items := e.Menu.Items()
		e.Menu = NewMenu([]*MenuItem{items[3], items[0], {Label: "Copy again", Submenu: []*MenuItem{items[3]}}})
	}
	if r := ask(); r.Menu == nil || !sameInts(natives(r.Menu.Items), []int{4, 1, 0}) {
		t.Errorf("an item twice: answer %+v, items %v", r, natives(r.Menu.Items))
	}
}

// What the page's contextmenu event recorded fills in what the backend
// does not tell, for the menu it opened: at the same point, and the page's
// zoom.
func TestContextMenuRecord(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	var got ContextMenuEvent
	w.Page().OnContextMenu(func(e *ContextMenuEvent) { got = *e })
	record := `{"t":"context-menu","x":10,"y":20,"l":"https://example.com/a","lt":"A link","i":"https://example.com/i.png","s":"chosen","e":true}`
	ask := func(m *platform.ContextMenu) ContextMenuEvent {
		got = ContextMenuEvent{}
		m.Items = webViewMenu()
		askContextMenu(fw, m)
		return got
	}

	page(fw, record)
	e := ask(&platform.ContextMenu{Pointer: true, X: 11, Y: 19, Target: platform.ContextMenuTarget{LinkURL: "https://example.com/told"}})
	if e.LinkURL != "https://example.com/told" || e.LinkText != "A link" || e.ImageURL != "https://example.com/i.png" || e.SelectionText != "chosen" || !e.Editable {
		t.Errorf("at the point: %+v", e)
	}
	// The record serves one menu.
	if e := ask(&platform.ContextMenu{}); e.SelectionText != "" {
		t.Errorf("a record served twice: %+v", e)
	}

	// Another point: the event of a frame's, which the page did not hear.
	page(fw, record)
	if e := ask(&platform.ContextMenu{Pointer: true, X: 200, Y: 20}); e.LinkURL != "" || e.SelectionText != "" || e.Editable {
		t.Errorf("at another point: %+v", e)
	}

	// The page's zoom makes its CSS pixels larger.
	onMain(func() { fw.SetZoom(2) })
	page(fw, record)
	if e := ask(&platform.ContextMenu{Pointer: true, X: 20, Y: 40}); e.SelectionText != "chosen" {
		t.Errorf("zoomed: %+v", e)
	}
	onMain(func() { fw.SetZoom(1) })

	// The keyboard's menu takes the latest record.
	page(fw, record)
	if e := ask(&platform.ContextMenu{}); e.LinkURL != "https://example.com/a" {
		t.Errorf("the keyboard's: %+v", e)
	}

	// A backend that told all takes nothing from it.
	page(fw, record)
	if e := ask(&platform.ContextMenu{Complete: true}); e.LinkURL != "" || e.SelectionText != "" {
		t.Errorf("complete: %+v", e)
	}

	// A new page drops the record of the one before.
	page(fw, record)
	onMain(func() { fw.H.NavigationCommitted("about:blank") })
	if e := ask(&platform.ContextMenu{}); e.LinkURL != "" {
		t.Errorf("after navigating: %+v", e)
	}
}

// Without a listener the page's records are not kept.
func TestContextMenuRecordUnheard(t *testing.T) {
	w, fw := testWindow(t, WindowOptions{})
	page(fw, `{"t":"context-menu","x":1,"y":1,"s":"old"}`)
	var got string
	w.Page().OnContextMenu(func(e *ContextMenuEvent) { got = e.SelectionText })
	askContextMenu(fw, &platform.ContextMenu{Items: webViewMenu()})
	if got != "" {
		t.Errorf("a record made with no listener served: %q", got)
	}
}

// The app's items of a web view's page in native UI tell its window.
func TestWebViewContextMenu(t *testing.T) {
	show := true
	w, v, _, fv := webViewWindow(t, &show)
	clicked := make(chan *Window, 1)
	v.Page().OnContextMenu(func(e *ContextMenuEvent) {
		e.Menu.Append(&MenuItem{Label: "Mine", Click: func(_ *MenuItem, w *Window) { clicked <- w }})
	})
	r := askContextMenu(fv.Window, &platform.ContextMenu{Items: webViewMenu()})
	if r.Menu == nil {
		t.Fatalf("answer = %+v", r)
	}
	onMain(func() { fv.H.MenuItemClicked(r.Menu.Items[len(r.Menu.Items)-1].ID) })
	select {
	case cw := <-clicked:
		if cw != w {
			t.Errorf("Click got %v, want the web view's window", cw)
		}
	default:
		t.Error("the app's item chosen ran no Click")
	}
}

// Menu.Remove removes items by ID from a menu and its submenus.
func TestMenuRemove(t *testing.T) {
	m := NewMenu([]*MenuItem{
		{ID: "a", Label: "A"},
		{Label: "Sub", Submenu: []*MenuItem{{ID: "b", Label: "B"}, {ID: "c", Label: "C"}}},
		{Label: "No ID"},
	})
	m.Remove("a", "b", "")
	if m.ItemByID("a") != nil || m.ItemByID("b") != nil || m.ItemByID("c") == nil {
		t.Error("Remove removed the wrong items")
	}
	if items := m.Items(); len(items) != 2 || items[1].Label != "No ID" {
		t.Errorf("items left = %d", len(items))
	}
}

// TestTidySeparators drops the separators that would start or end a menu
// shown, or follow another, hidden items between them aside.
func TestTidySeparators(t *testing.T) {
	sep := func() *platform.MenuItem { return &platform.MenuItem{Type: platform.MenuItemSeparator, Visible: true} }
	item := func(label string, visible bool) *platform.MenuItem {
		return &platform.MenuItem{Label: label, Visible: visible}
	}
	m := &platform.Menu{Items: []*platform.MenuItem{
		sep(), item("Back", false), sep(), item("Copy", true), sep(), item("Hidden", false), sep(),
		item("Paste", true), {Label: "More", Visible: true, Type: platform.MenuItemSubmenu,
			Submenu: &platform.Menu{Items: []*platform.MenuItem{sep(), item("Speak", true), sep()}}},
		sep(), item("Gone", false),
	}}
	m.TidySeparators()
	var got []string
	var list func(prefix string, items []*platform.MenuItem)
	list = func(prefix string, items []*platform.MenuItem) {
		for _, it := range items {
			if it.Type == platform.MenuItemSeparator {
				got = append(got, prefix+"-")
				continue
			}
			got = append(got, prefix+it.Label)
			if it.Submenu != nil {
				list(prefix+it.Label+"/", it.Submenu.Items)
			}
		}
	}
	list("", m.Items)
	want := []string{"Back", "Copy", "-", "Hidden", "Paste", "More", "More/Speak", "Gone"}
	if !slices.Equal(got, want) {
		t.Errorf("tidied menu %v, want %v", got, want)
	}
}
