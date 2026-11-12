// Package whatsnew presents a release's highlights over the settings
// window: a recording of mock UI on the photograph above the slide's title
// and text, with Previous, Next and Skip under it. It shows the slides the
// store lists (entities/whatsnew picks them for the versions an update
// crossed) and reports when the person finishes or skips them.
package whatsnew

import (
	"time"

	"arion.local/desktop/ui/shared/assets"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/AstraFoundry/design/packages/mygo/sunbeam"
	"github.com/egoist/mygo/ui"
)

// Width is the dialog's width and its stage's; StageHeight is the stage's
// height in a window with room for it. A shorter window gets a shorter
// stage with the recording scaled to fit; the text keeps its size.
const (
	Width       = 560
	StageHeight = 300

	// margin keeps the dialog off the window's edges; below is the room
	// the text and buttons under the stage take.
	margin   = 20
	below    = 196
	minStage = 160
	radius   = theme.RadiusXL + 4
)

// Dialog keeps which slide shows and since when, so each slide's recording
// starts from its beginning.
type Dialog struct {
	seq, index int
	since      time.Time
	open       bool
}

// Reset forgets the presentation, as when its window closes: the next one
// starts at its first slide.
func (d *Dialog) Reset() { d.open = false }

// Render shows the store's slides over the window it is built in. It does
// nothing while there are none.
func (d *Dialog) Render(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	news := store.State.WhatsNew
	if len(news.Slides) == 0 {
		d.open = false
		return
	}
	if !d.open || news.Seq != d.seq {
		d.open, d.seq = true, news.Seq
		d.show(c, 0)
	}
	d.index = min(d.index, len(news.Slides)-1)
	messages := store.Copy()
	slide := news.Slides[d.index]
	scene := scenes[slide.ID]
	last := d.index == len(news.Slides)-1

	_, height := c.Size()
	stage := min(StageHeight, max(height-2*margin-below, minStage))
	scale := stage / StageHeight

	open := true
	ui.DialogBase(c, &open, func(backdrop, panel ui.Element) {
		backdrop.Background(tokens.Scrim.Alpha(0.45)).Transition(ui.ElementTransition{Animation: &theme.Quick, Enter: &ui.Motion{}})
		kit.Popover(c, panel, radius).Width(Width).Label(messages.Text(copy.WhatsNewTitle)).
			Transition(ui.ElementTransition{Animation: &theme.Page, Enter: &ui.Motion{Y: 12, Scale: 0.97}})
		sunbeam.Layer(sunbeam.Raised, func() {
			// The photograph fills the stage; the recording plays on it,
			// scaled to the stage's height and centered. Each layer carries
			// the top corners itself.
			corner := float32(radius - 1)
			ui.Column(c.Key("stage-"+slide.ID)).Height(stage).Clip().Radius(corner, corner, 0, 0).
				Background(tokens.Scrim).Role(ui.RoleImage).Label(messages.Text(scene.label)).
				Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{}}).Children(func() {
				ui.Image(c, assets.Backdrop).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Cover).Radius(corner, corner, 0, 0)
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Radius(corner, corner, 0, 0).
					LinearGradient(ui.LinearGradient{From: tokens.Scrim.Alpha(0.12), To: tokens.Scrim.Alpha(0.4), Angle: 180})
				ui.Column(c).Absolute().Top(0).Left(Width*(1-scale)/2).Size(Width, StageHeight).TransformOrigin(0, 0).Scale(scale, scale).Children(func() {
					scene.play(c, tokens, messages, d.since)
				})
			})
			ui.Column(c).Padding(20, 24, 20, 24).Gap(18).Children(func() {
				ui.Column(c.Key("text-" + slide.ID)).Gap(6).MinHeight(92).
					Transition(ui.ElementTransition{Animation: &theme.Settle, Enter: &ui.Motion{Y: 6}}).Children(func() {
					ui.Text(c, messages.WhatsNewVersion(slide.Version)).FontSize(theme.Caption).FontWeight(500).TextColor(tokens.Muted)
					ui.Text(c, messages.Text(scene.title)).FontSize(theme.Heading + 2).FontWeight(600)
					ui.Text(c, messages.Text(scene.body)).FontSize(theme.Body).LineHeight(1.45).TextColor(tokens.Muted)
				})
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if len(news.Slides) > 1 {
						dots(c, tokens, messages, d.index, len(news.Slides))
					}
					ui.Spacer(c)
					if !last && kit.Button(c.Key("skip"), tokens, kit.Ghost, messages.Text(copy.Skip), true).Clicked() {
						open = false
					}
					if d.index > 0 && kit.Button(c.Key("previous"), tokens, kit.Outline, messages.Text(copy.WhatsNewPrevious), true).Clicked() {
						d.show(c, d.index-1)
					}
					next := copy.WhatsNewNext
					if last {
						next = copy.Done
					}
					if kit.Button(c.Key("next"), tokens, kit.Primary, messages.Text(next), true).AutoFocus().Clicked() {
						if last {
							open = false
						} else {
							d.show(c, d.index+1)
						}
					}
				})
			})
		})
		if backdrop.OverlayShortcut(0, ui.KeyRight) && !last {
			d.show(c, d.index+1)
		}
		if backdrop.OverlayShortcut(0, ui.KeyLeft) && d.index > 0 {
			d.show(c, d.index-1)
		}
	})
	if !open {
		d.open = false
		store.Actions.DismissWhatsNew()
	}
}

// show moves to slide index and starts its recording over.
func (d *Dialog) show(c *ui.Context, index int) {
	d.index, d.since = index, c.Now()
}

// dots mark the slide among the others; the current one is the accent's
// line color and wider.
func dots(c *ui.Context, tokens theme.Tokens, messages copy.Catalog, index, total int) {
	ui.Row(c).Gap(5).AlignItems(ui.Center).Role(ui.RoleImage).Label(messages.Step(index+1, total)).Children(func() {
		for i := range total {
			width, ink := float32(6), tokens.Text.Alpha(0.18)
			if i == index {
				width, ink = 16, tokens.AccentInk
			}
			ui.Box(c.Key(i)).Size(width, 6).Radius(3).Background(ink).Transition(ui.ElementTransition{Animation: &theme.Settle, Size: true})
		}
	})
}
