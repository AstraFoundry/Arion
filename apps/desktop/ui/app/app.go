// Package app composes the native UI entry points: one view per window.
package app

import (
	"arion.local/desktop/ui/pages/clipboard"
	"arion.local/desktop/ui/pages/clipboardaccess"
	"arion.local/desktop/ui/pages/feedback"
	"arion.local/desktop/ui/pages/guide"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/overlay"
	"arion.local/desktop/ui/pages/selection"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/pages/trayhint"
	"arion.local/desktop/ui/pages/updatereminder"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/typography"
	"github.com/egoist/mygo/ui"
)

func Clipboard(store runtime.Store) View {
	page := &clipboard.Page{}
	return View{Content: view(func(c *ui.Context) { page.Render(c, store) }).WithClipboardWriter(func(text string) { store.Actions.CopyText(text) }), Close: func() {}}
}

const ClipboardWidth = clipboard.Width

const ClipboardAccessWidth = clipboardaccess.Width
const ClipboardAccessHeight = clipboardaccess.Height

func ClipboardAccess(store runtime.Store, resize func(int)) View {
	return View{Content: view(func(c *ui.Context) {
		height := clipboardaccess.Render(c, store)
		if resize != nil && height > 0 {
			resize(height)
		}
	}), Close: func() {}}
}

func ClipboardHeight(items []runtime.Clip) int { return clipboard.Height(items) }

func PanelHeightFor(store runtime.Store) int { return menubar.HeightFor(store) }

// Window sizes in DIPs, owned by the pages that lay them out.
const (
	PanelWidth           = menubar.Width
	PanelHeight          = menubar.Height
	OnboardingWidth      = onboarding.Width
	OnboardingHeight     = onboarding.Height
	SettingsWidth        = 760
	SettingsHeight       = 560
	OverlayWidth         = 720
	OverlayHeight        = 180
	GuideHeight          = guide.Height
	TrayHintWidth        = trayhint.Width
	TrayHintHeight       = trayhint.Height
	UpdateReminderWidth  = updatereminder.Width
	UpdateReminderHeight = updatereminder.Height
	FeedbackWidth        = feedback.Width
	FeedbackHeight       = feedback.Height
)

// View is a window's content and the hook releasing what it holds.
type View struct {
	Content *ui.Content
	Close   func()
}

func Overlay(store runtime.Store) View {
	page := overlay.NewPage()
	return View{Content: view(func(c *ui.Context) { page.Render(c, store) }), Close: func() {}}
}

func Guide(store runtime.Store) View {
	return View{Content: view(func(c *ui.Context) { guide.Render(c, store) }), Close: func() {}}
}

func TrayHint(store runtime.Store) View {
	return View{Content: view(func(c *ui.Context) { trayhint.Render(c, store) }), Close: func() {}}
}

func Panel(store runtime.Store) View {
	page := &menubar.Panel{}
	return View{Content: view(func(c *ui.Context) { page.Render(c, store) }).WithClipboardWriter(func(text string) { store.Actions.CopyText(text) }), Close: func() {}}
}

func Settings(store runtime.Store) View {
	page := settings.NewPage()
	return View{Content: view(func(c *ui.Context) { page.Render(c, store) }), Close: func() { page.Close(store) }}
}

func Onboarding(store runtime.Store) View {
	page := onboarding.NewPage()
	return View{Content: view(func(c *ui.Context) { page.Render(c, store) }), Close: func() { page.Close(store) }}
}

// Feedback is the window that hands a problem report to a coding agent.
func Feedback(store runtime.Store) View {
	page := &feedback.Page{}
	return View{Content: view(func(c *ui.Context) { page.Render(c, store) }).WithClipboardWriter(func(text string) { store.Actions.CopyText(text) }), Close: func() {}}
}

// SelectionWidth is the selection bar's width as it opens, before the bar
// has measured itself. SelectionHeight is the bar window's height, which
// its menus do not change: each shows in a popup window beside it,
// SelectionGap away.
const (
	SelectionWidth  = selection.Width
	SelectionHeight = selection.Height
	SelectionGap    = selection.Gap
)

// SelectionPopup is the menu or failure line the selection bar shows in a
// window beside it.
type SelectionPopup = selection.Popup

// SelectionViews are the selection bar's window and the popup windows
// beside it, which share its menus.
type SelectionViews struct {
	// Bar is the bar's window.
	Bar View
	// Popup is a window for the popup key. above reports whether the
	// window opened above the bar; gone runs once the bar wants another
	// popup and this one has left, when its window may close.
	Popup func(key string, above func() bool, gone func()) View
	// Dismiss closes the popup key that the system closed, as on a press
	// on the control that opened it.
	Dismiss func(key string)
}

// Selection is the bar over another app's selected text. After each frame
// of the bar or a popup, sync receives the bar's width (0 before it has
// been laid out), the popup it wants and a revision that changes whenever
// what the bar or a popup shows changes.
func Selection(store runtime.Store, sync func(width int, popup SelectionPopup, revision int)) SelectionViews {
	page := &selection.Page{}
	report := func() {
		if sync != nil {
			sync(page.Width(), page.Want(), page.Revision())
		}
	}
	return SelectionViews{
		Bar: View{Content: view(func(c *ui.Context) {
			page.Render(c, store)
			report()
		}), Close: func() {}},
		Popup: func(key string, above func() bool, gone func()) View {
			surface := &selection.Surface{Key: key}
			left := false
			return View{Content: view(func(c *ui.Context) {
				surface.Above = above()
				if page.RenderPopup(c, store, surface) && !left {
					left = true
					gone()
				}
				report()
			}), Close: func() {}}
		},
		Dismiss: page.Dismiss,
	}
}

// GuideHeightFor sizes each System Settings guide for its content.
func GuideHeightFor(pane runtime.GuidePane) int { return guide.HeightFor(pane) }

// UpdateReminder is the separate passive reminder; bootstrap owns its window.
func UpdateReminder(store runtime.Store) View {
	return View{Content: view(func(c *ui.Context) { updatereminder.Render(c, store) }), Close: func() {}}
}

const ClipboardPreviewWidth = clipboard.PreviewWidth
const ClipboardPreviewHeight = clipboard.PreviewHeight

func ClipboardPreview(item runtime.Clip, bitmap *ui.Bitmap) View {
	return View{Content: view(func(c *ui.Context) { clipboard.Preview(c, item, bitmap) }), Close: func() {}}
}

// view applies the platform interface family to every window and its overlays.
func view(render func(*ui.Context)) *ui.Content {
	return ui.View(interfaceView(render))
}

func interfaceView(render func(*ui.Context)) func(*ui.Context) {
	return func(c *ui.Context) {
		theme := *c.Theme()
		theme.Font = typography.Family
		c.SetTheme(&theme)
		render(c)
	}
}
