// Permissionflow lists the permissions the system keeps for apps, with
// their status, and asks for them: with the system's prompt, or by
// guiding the user to System Settings, where a panel under its window
// holds the app to drag into the list (plugins/permissionflow).
//
//	go run ./examples/permissionflow
//
// Run from a terminal, macOS gives the permissions to the terminal: build
// the app (go run ./cmd/mygo build examples/permissionflow) to see them
// given to it.
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/permissionflow"
	"github.com/egoist/mygo/ui"
)

// row is a permission the system has, and what is known of it.
type row struct {
	p      permissionflow.Permission
	status permissionflow.Status
	err    error
	busy   bool
}

// model is the state the window shows, and how it reaches the system,
// which the test replaces.
type model struct {
	rows    []*row
	request func(permissionflow.Permission)
	guide   func(permissionflow.Permission)
}

func (m *model) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Children(func() {
		ui.Column(c).Padding(16, 16, 8).Gap(4).Children(func() {
			ui.Text(c, "Permissions").FontSize(20).Bold()
			ui.Text(c, "Request asks with the system's prompt. Guide opens the system's settings, with the app to drag where they take it.").
				TextColor(t.TextMuted)
		})
		if len(m.rows) == 0 {
			ui.Column(c).Grow(1).Center().Children(func() {
				ui.Text(c, "This system keeps none of these permissions for apps.").TextColor(t.TextMuted)
			})
			return
		}
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(0, 16, 16).Gap(8).Children(func() {
				for _, r := range m.rows {
					m.rowView(c, r)
				}
			})
		})
	})
}

func (m *model) rowView(c *ui.Context, r *row) {
	t := c.Theme()
	status, color := r.status.String(), t.TextMuted
	switch {
	case r.err != nil:
		status, color = r.err.Error(), t.Danger
	case r.status == permissionflow.Granted:
		color = t.Success
	}
	ui.Row(c).Gap(12).Padding(10, 12).AlignItems(ui.Center).Radius(t.Radius).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, r.p.Title()).Bold()
			ui.Text(c, status).FontSize(12).TextColor(color)
		})
		done := r.status == permissionflow.Granted
		if ui.Button(c.Key(r.p+"-request"), "Request").Label("Request " + r.p.Title()).Disabled(done || r.busy).Clicked() {
			r.busy = true
			m.request(r.p)
		}
		if ui.PrimaryButton(c.Key(r.p+"-guide"), "Guide").Label("Guide to " + r.p.Title()).Disabled(done || r.busy).Clicked() {
			r.busy = true
			m.guide(r.p)
		}
	})
}

// refresh checks every permission again.
func (m *model) refresh() {
	for _, r := range m.rows {
		r.status, r.err = permissionflow.Check(r.p)
	}
}

func main() {
	m := &model{}
	for _, p := range permissionflow.All() {
		if _, err := permissionflow.Check(p); !errors.Is(err, errors.ErrUnsupported) {
			m.rows = append(m.rows, &row{p: p})
		}
	}
	var win *mygo.Window
	// done ends a request or a guide: it shows the status, or the error.
	done := func(p permissionflow.Permission, err error) {
		win.Update(func() {
			m.refresh()
			for _, r := range m.rows {
				if r.p == p {
					r.busy = false
					if err != nil && !errors.Is(err, context.DeadlineExceeded) {
						r.err = err
					}
				}
			}
		})
	}
	m.request = func(p permissionflow.Permission) {
		go func() {
			_, err := permissionflow.Request(context.Background(), p)
			done(p, err)
		}()
	}
	m.guide = func(p permissionflow.Permission) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			_, err := permissionflow.Guide(ctx, p, permissionflow.GuideOptions{})
			done(p, err)
		}()
	}
	mygo.App.WhenReady(func() {
		m.refresh()
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Permissions",
			Width:   520,
			Height:  640,
			Content: ui.View(m.view),
		})
		// The user changes them in the system's settings meanwhile.
		go func() {
			for range time.Tick(time.Second) {
				win.Update(m.refresh)
			}
		}()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
