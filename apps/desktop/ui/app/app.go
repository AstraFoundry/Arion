// Package app composes the native UI entry points: one view per window.
package app

import (
	"arion.local/desktop/ui/pages/guide"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/onboarding"
	"arion.local/desktop/ui/pages/overlay"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

// Window sizes in DIPs, owned by the pages that lay them out.
const (
	PanelWidth       = menubar.Width
	PanelHeight      = menubar.Height
	OnboardingWidth  = onboarding.Width
	OnboardingHeight = onboarding.Height
	SettingsWidth    = 760
	SettingsHeight   = 560
	OverlayWidth     = 720
	OverlayHeight    = 180
	GuideHeight      = guide.Height
)

// View is a window's content and the hook releasing what it holds.
type View struct {
	Content *ui.Content
	Close   func()
}

func Overlay(store runtime.Store) View {
	return View{Content: ui.View(func(c *ui.Context) { overlay.Render(c, store) }), Close: func() {}}
}

func Guide(store runtime.Store) View {
	return View{Content: ui.View(func(c *ui.Context) { guide.Render(c, store) }), Close: func() {}}
}

func Panel(store runtime.Store) View {
	page := &menubar.Panel{}
	return View{Content: ui.View(func(c *ui.Context) { page.Render(c, store) }), Close: func() {}}
}

func Settings(store runtime.Store) View {
	page := settings.NewPage()
	return View{Content: ui.View(func(c *ui.Context) { page.Render(c, store) }), Close: func() { page.Close(store) }}
}

func Onboarding(store runtime.Store) View {
	page := onboarding.NewPage()
	return View{Content: ui.View(func(c *ui.Context) { page.Render(c, store) }), Close: func() { page.Close(store) }}
}
