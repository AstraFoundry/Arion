// Package app composes the native UI entry point.
package app

import (
	"arion.local/desktop/ui/pages/home"
	"github.com/egoist/mygo/ui"
)

func View() *ui.Content { return ui.View(home.Render) }
