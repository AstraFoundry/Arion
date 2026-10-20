package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Icons of the status demos.
var (
	statusInbox  = icon(`<path d="M22 12h-6l-2 3h-4l-2-3H2"/><path d="M5.5 5.1 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.5-6.9A2 2 0 0 0 16.8 4H7.2a2 2 0 0 0-1.7 1.1z"/>`)
	statusSend   = icon(`<path d="m22 2-7 20-4-9-9-4z"/><path d="M22 2 11 13"/>`)
	statusDraft  = icon(`<path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"/>`)
	statusLoader = icon(`<path d="M21 12a9 9 0 1 1-6.2-8.6"/>`)
	statusHeart  = icon(`<path d="M19 14c1.5-1.5 3-3.2 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.8 0-3 .5-4.5 2-1.5-1.5-2.7-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4 3 5.5l7 7z"/>`)
)

// statusArt is an illustration in its own colors, with currentColor for
// its ground.
var statusArt = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 80">
<defs><linearGradient id="sky" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#38BDF8"/><stop offset="1" stop-color="#E0F2FE"/></linearGradient>
<clipPath id="card"><rect width="120" height="80" rx="10"/></clipPath></defs>
<g clip-path="url(#card)">
<rect width="120" height="80" fill="url(#sky)"/>
<circle cx="88" cy="24" r="10" fill="#FACC15"/>
<path d="M0 62 30 30l22 22 14-12 26 22H0z" fill="#16A34A"/>
<path d="M30 30l8 8-8 4-8-4z" fill="#F0FDF4"/>
<rect y="62" width="120" height="18" fill="currentColor" opacity=".25"/>
</g>
</svg>`))

// statusMix mixes a and b, t of the way to b.
func statusMix(a, b color.RGBA, t float64) color.RGBA {
	t = max(0, min(1, t))
	m := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t + 0.5) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

// statusPhoto draws a sunset over hills, a photo of 320×180 pixels.
func statusPhoto() *image.RGBA {
	const w, h = 320, 180
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	top, mid, low := color.RGBA{30, 41, 99, 255}, color.RGBA{219, 39, 119, 255}, color.RGBA{251, 146, 60, 255}
	sun := color.RGBA{254, 240, 138, 255}
	far, near := color.RGBA{88, 28, 135, 255}, color.RGBA{30, 27, 75, 255}
	for y := range h {
		for x := range w {
			fy := float64(y) / 120
			c := statusMix(top, mid, fy*1.6)
			if fy > 0.6 {
				c = statusMix(mid, low, (fy-0.6)/0.4)
			}
			// The sun, with a soft edge.
			d := math.Hypot(float64(x)-210, float64(y)-112) - 30
			c = statusMix(c, sun, 0.5-d)
			if glow := 1 - math.Hypot(float64(x)-210, float64(y)-112)/110; glow > 0 {
				c = statusMix(c, sun, glow*glow*0.35)
			}
			// Two hills, the far one first.
			fx := float64(x)
			if hy := 128 + 14*math.Sin(fx/38) + 6*math.Sin(fx/13); float64(y) > hy {
				c = statusMix(c, far, float64(y)-hy)
			}
			if hy := 148 + 12*math.Sin(fx/52+2) + 4*math.Sin(fx/17); float64(y) > hy {
				c = statusMix(c, near, float64(y)-hy)
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// statusPortrait draws a person's silhouette on a gradient, 96 pixels
// square.
func statusPortrait() *image.RGBA {
	const n = 96
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	a, b := color.RGBA{45, 212, 191, 255}, color.RGBA{59, 130, 246, 255}
	skin := color.RGBA{255, 237, 213, 255}
	for y := range n {
		for x := range n {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			c := statusMix(a, b, (fx+fy)/(2*n))
			head := 20 - math.Hypot(fx-48, fy-38)
			body := 34 - math.Hypot((fx-48)*0.8, fy-100)
			c = statusMix(c, skin, max(head, body)+0.5)
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// statusLogo draws a logo, a ring on a diagonal gradient, 128 pixels
// square.
func statusLogo() *image.RGBA {
	const n = 128
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	a, b := color.RGBA{99, 102, 241, 255}, color.RGBA{236, 72, 153, 255}
	white := color.RGBA{255, 255, 255, 255}
	for y := range n {
		for x := range n {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			c := statusMix(a, b, (fx+fy)/(2*n))
			ring := 9 - math.Abs(math.Hypot(fx-64, fy-64)-34)
			c = statusMix(c, white, ring+0.5)
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

var (
	statusPhotoBitmap    = ui.NewBitmap(statusPhoto())
	statusPortraitBitmap = ui.NewBitmap(statusPortrait())
	statusLogoBitmap     = ui.NewBitmap(statusLogo())
)

func init() {
	done, total := 2, 5
	demo("progress", func(c *ui.Context) {
		ui.Column(c).Width(320).Gap(8).Children(func() {
			ui.Text(c, fmt.Sprintf("Uploaded %d of %d files", done, total)).FontSize(12)
			ui.Progress(c, float64(done)/float64(total)).Label("Uploading")
			ui.Text(c, "Connecting to the server…").FontSize(12).MarginY(4)
			ui.Progress(c, -1).Label("Connecting")
		})
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Upload a file").Disabled(done == total).Clicked() {
				done++
			}
			if ui.Button(c, "Reset").Disabled(done == 0).Clicked() {
				done = 0
			}
		})
	})

	loading := true
	demo("spinner", func(c *ui.Context) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Height(24).Children(func() {
			if loading {
				ui.Spinner(c).Label("Loading")
				ui.Text(c, "Loading messages…")
			} else {
				ui.Text(c, "12 messages")
			}
		})
		label := "Stop"
		if !loading {
			label = "Reload"
		}
		if ui.Button(c, label).Clicked() {
			loading = !loading
		}
	})

	used, battery := 430.0, 64.0
	demo("meter", func(c *ui.Context) {
		const size = 500
		ui.Column(c).Width(320).Gap(6).Children(func() {
			ui.Row(c).Children(func() {
				ui.Text(c, "Disk").FontSize(12)
				ui.Spacer(c)
				ui.Text(c, fmt.Sprintf("%.0f of %d GB", used, size)).FontSize(12).TextColor(c.Theme().TextMuted)
			})
			ui.Meter(c, used, 0, size, &ui.MeterLevels{Warning: 0.8 * size, Critical: 0.95 * size}).Label("Disk")
			ui.Slider(c, &used, 0, size).Label("Disk used")
			ui.Row(c).MarginY(4).Children(func() {
				ui.Text(c, "Battery").FontSize(12)
				ui.Spacer(c)
				ui.Text(c, fmt.Sprintf("%.0f%%", battery)).FontSize(12).TextColor(c.Theme().TextMuted)
			})
			ui.Meter(c, battery, 0, 100, &ui.MeterLevels{Warning: 20, Critical: 10}).Label("Battery")
			ui.Slider(c, &battery, 0, 100).Label("Battery charge")
		})
	})

	mailbox, unread := "drafts", 3
	demo("badge", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
			if ui.Sidebar(c, &mailbox, func() {
				ui.SidebarSection(c, "Mailboxes", nil, func() {
					ui.SidebarItem(c, "inbox", statusInbox, "Inbox").Children(func() {
						if unread > 0 {
							ui.Badge(c, fmt.Sprint(unread))
						}
					})
					ui.SidebarItem(c, "drafts", statusDraft, "Drafts").Children(func() {
						ui.Badge(c, "New").Background(t.Accent).TextColor(t.AccentText)
					})
					ui.SidebarItem(c, "sent", statusSend, "Sent")
				})
			}).Width(200).Height(130).Radius(8).Background(t.Surface).Changed() && mailbox == "inbox" {
				unread = 0
			}
			ui.Column(c).Gap(8).AlignItems(ui.Start).Children(func() {
				if ui.Button(c, "Receive a message").Clicked() {
					unread++
				}
				if unread > 0 {
					status(c, fmt.Sprintf("%d unread: open the inbox to read them", unread))
				} else {
					status(c, "No unread messages")
				}
			})
		})
	})

	fit, gray := 0, false
	fits := []ui.Fit{ui.Contain, ui.Cover, ui.FillBox, ui.ScaleDown, ui.NaturalSize}
	demo("image", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(20).AlignItems(ui.Center).Children(func() {
			ui.Image(c, statusLogoBitmap).Size(64, 64).Fit(ui.Contain).Radius(12)
			photo := ui.Image(c, statusPhotoBitmap).Size(200, 130).Fit(fits[fit]).Radius(8).Background(t.Surface)
			if gray {
				photo.Grayscale()
			}
			ui.Image(c, statusArt).Width(120)
		})
		ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &fit, "Contain", "Cover", "Fill box", "Scale down", "Natural size").Label("Fit")
			ui.Checkbox(c, &gray, "Grayscale")
		})
	})

	demo("icon", func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconSave)
			ui.Text(c, "Save")
		})
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconSave).FontSize(24).TextColor(t.Danger)
			ui.Icon(c, iconStar).FontSize(24).TextColor(t.Warning)
			ui.Icon(c, statusHeart).FontSize(24).TextColor(t.Accent)
			ui.Icon(c, iconBell).FontSize(24).TextColor(t.Success)
		})
		ui.Row(c).Gap(8).Children(func() {
			ui.PrimaryButton(c, "").Children(func() {
				ui.Icon(c, iconSave)
				ui.Text(c, "Save").SingleLine()
			})
			ui.Button(c, "").Children(func() {
				spin := ui.Icon(c, statusLoader)
				spin.Rotate(spin.Loop("spin", time.Second, ui.Linear) * 360)
				ui.Text(c, "Syncing").SingleLine()
			})
		})
	})

	name := "Ada Lovelace"
	people := []string{"Grace Hopper", "Alan Turing", "Margaret Hamilton", "Linus"}
	demo("avatar", func(c *ui.Context) {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Avatar(c, "Katherine Johnson", statusPortraitBitmap)
			for _, p := range people {
				ui.Avatar(c, p, nil)
			}
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Avatar(c, name, nil)
			ui.TextInput(c, &name).Width(200).Placeholder("A name").Label("Name")
		})
	})
}
