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

const SelectionWidth = selection.Width

// SelectionHeightFor sizes the selection popover for the current offer.
func SelectionHeightFor(store runtime.Store) int { return selection.Height(store) }

// Selection is the popover over another app's selected text. resize
// receives the height the menu needs, which grows with a failure line.
func Selection(store runtime.Store, resize func(int)) View {
	page := &selection.Page{}
	return View{Content: view(func(c *ui.Context) {
		height := page.Render(c, store)
		if resize != nil {
			resize(height)
		}
	}), Close: func() {}}
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
