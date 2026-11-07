// Package feedback is the window that hands a problem report to a coding
// agent: the recording of how it goes beside the prompt to copy.
package feedback

import (
	goruntime "runtime"
	"time"

	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"arion.local/desktop/ui/shared/typography"
	"arion.local/desktop/ui/widgets/agentdemo"
	"github.com/egoist/mygo/ui"
)

// Width and Height size the window; the illustration takes its own stage
// on the left, and Inset is the margin of the column beside it.
const (
	Width  = 820
	Height = agentdemo.Height
	Inset  = 40
)

// confirmed is how long the copy button says the prompt was copied.
const confirmed = 1800 * time.Millisecond

// Page keeps when the prompt was last copied.
type Page struct{ copied time.Time }

// platforms names the operating systems Arion ships for.
var platforms = map[string]string{"darwin": "macOS", "windows": "Windows"}

// Prompt is what the person pastes into their agent: where the feedback
// skill is, and the build and platform it is about.
func Prompt(store runtime.Store) string {
	name, ok := platforms[goruntime.GOOS]
	if !ok {
		name = goruntime.GOOS
	}
	return store.Copy().FeedbackPrompt(config.FeedbackSkillURL, store.State.Version, name+" "+goruntime.GOARCH)
}

// Render draws the recording beside the prompt.
func (p *Page) Render(c *ui.Context, store runtime.Store) {
	messages := store.Copy()
	tokens := theme.Apply(c)
	prompt := Prompt(store)
	top := max(c.TitleBar().Height, 28) + 16
	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(tokens.Canvas).Children(func() {
		ui.Column(c).Width(agentdemo.Width).Children(func() {
			agentdemo.Render(c, tokens, messages, prompt)
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(top).DragWindow()
		})
		ui.Column(c).Grow(1).Children(func() {
			ui.Box(c).Height(top).DragWindow()
			ui.Column(c).Grow(1).Padding(0, Inset, Inset, Inset).Gap(20).Justify(ui.Center).Children(func() {
				ui.Column(c).Gap(8).Children(func() {
					ui.Text(c, messages.Text(copy.FeedbackTitle)).FontSize(theme.Title + 4).FontWeight(650)
					ui.Text(c, messages.Text(copy.FeedbackBody)).FontSize(theme.Body).TextColor(tokens.Muted)
				})
				p.prompt(c, store, tokens, messages, prompt)
				ui.Row(c).Children(func() {
					ui.Link(c, messages.Text(copy.FeedbackIssueYourself), config.IssuesURL).FontSize(theme.Caption)
				})
			})
		})
	})
}

// prompt shows the prompt with the button that copies it.
func (p *Page) prompt(c *ui.Context, store runtime.Store, tokens theme.Tokens, messages copy.Catalog, prompt string) {
	kit.Card(c, tokens).Padding(14).Gap(12).Children(func() {
		ui.Text(c, prompt).Font(typography.Mono).FontSize(12).LineHeight(1.5).Selectable()
		ui.Row(c).Children(func() {
			ui.Spacer(c)
			done := !p.copied.IsZero() && c.Now().Sub(p.copied) < confirmed
			if done {
				// Keep frames coming until the confirmation reverts.
				c.AnimationFrame()
			}
			label, icon := messages.Text(copy.Copy), icons.Copy
			if done {
				label, icon = messages.Text(copy.Copied), icons.Check
			}
			button := ui.ButtonBase(c.Key("copy")).Height(32).Padding(0, 12).Gap(6).Radius(theme.RadiusControl).AlignItems(ui.Center)
			kit.Style(button, tokens, kit.Primary, true)
			button.Children(func() {
				ui.Row(c.Key(label)).Gap(6).AlignItems(ui.Center).Transition(ui.ElementTransition{Animation: &theme.Pop, Enter: &ui.Motion{Scale: 0.9}}).Children(func() {
					ui.Icon(c, icon).FontSize(13)
					ui.Text(c, label).FontSize(theme.Body).FontWeight(500).SingleLine()
				})
			})
			if button.Clicked() && store.Actions.CopyText(prompt) {
				p.copied = c.Now()
			}
		})
	})
}
