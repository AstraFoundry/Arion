// Package bootstrap constructs the production native desktop window.
package bootstrap

import (
	"arion.local/desktop/ui/app"
	"arion.local/desktop/ui/shared/copy"
	"github.com/egoist/mygo"
)

func WindowOptions() mygo.WindowOptions {
	return mygo.WindowOptions{
		Title: copy.Title, Width: 480, Height: 300,
		MinWidth: 360, MinHeight: 240, StateKey: "main", Content: app.View(),
	}
}
