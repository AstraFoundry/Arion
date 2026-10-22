// Package cloud connects a cloud model for polishing, one page at a time:
// the providers, then the chosen provider's sign-in or API key, then its
// model. Secrets never enter the UI state; a pasted key leaves the
// navigation state as soon as it is submitted.
package cloud

import (
	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Nav is where the pages are: the provider list while Provider is empty,
// else that provider's page. The page that shows them keeps it.
type Nav struct {
	Provider string
	// Key is the API key being entered on a provider's page.
	Key string
}

// Back returns from a provider's page to the list; false at the list.
func (n *Nav) Back() bool {
	if n.Provider == "" {
		return false
	}
	n.Provider, n.Key = "", ""
	return true
}

// Title names the current page.
func Title(store runtime.Store, nav *Nav) string {
	if nav.Provider != "" {
		return config.CloudProviders[nav.Provider]
	}
	return store.Copy().Text(copy.CloudTitle)
}

// providers is the list order: the subscription first, then API keys.
func providers() []string {
	return append([]string{runtime.CloudChatGPT}, runtime.CloudKeyProviders...)
}

// Mark draws a provider's LobeHub mark at size DIPs: monochrome marks in
// text ink, Gemini and DeepSeek in their colors.
func Mark(c *ui.Context, tokens theme.Tokens, provider string, size float32) {
	switch provider {
	case runtime.CloudGoogle:
		ui.Image(c, icons.Gemini).Size(size, size)
	case runtime.CloudDeepSeek:
		ui.Image(c, icons.DeepSeek).Size(size, size)
	default:
		mark := map[string]*ui.SVG{runtime.CloudChatGPT: icons.OpenAI, runtime.CloudOpenAI: icons.OpenAI,
			runtime.CloudAnthropic: icons.Anthropic, runtime.CloudOpenRouter: icons.OpenRouter}[provider]
		ui.Icon(c, mark).FontSize(size).TextColor(tokens.Text)
	}
}

// Check lists the connected provider's models once while a page needs them.
func Check(store runtime.Store) {
	cl := store.State.Cloud
	if cl.Phase == runtime.CloudConnected && cl.Models == nil && !cl.Listing && cl.Failure == "" {
		store.Actions.ListCloudModels()
	}
}

// Summary is the polish card's row that opens the pages: the connected
// provider and model, or that nothing is connected. It reports a click.
func Summary(c *ui.Context, store runtime.Store, tokens theme.Tokens) bool {
	messages := store.Copy()
	cl := store.State.Cloud
	detail := messages.Text(copy.CloudNotConnected)
	if cl.Phase == runtime.CloudConnected {
		detail = config.CloudProviders[cl.Provider]
		if m := store.State.Preferences.CloudModel; m != "" {
			detail += " · " + m
		}
	}
	kit.Divider(c, tokens)
	return link(c, tokens, "cloud-summary", messages.Text(copy.CloudTitle), detail, func() {
		if cl.Phase == runtime.CloudConnected {
			Mark(c, tokens, cl.Provider, 16)
		}
	})
}

// link is a row that opens a page: title, detail, an optional leading mark
// and a chevron.
func link(c *ui.Context, tokens theme.Tokens, key, title, detail string, lead func()) bool {
	row := ui.ButtonBase(c.Key(key)).Gap(12).Padding(12, 16).AlignItems(ui.Center).Label(title)
	if row.Hovered() {
		row.Background(tokens.AccentSoft.Alpha(0.5).Over(tokens.Card))
	}
	row.Children(func() {
		if lead != nil {
			lead()
		}
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(theme.Body).FontWeight(500)
			if detail != "" {
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted).SingleLine()
			}
		})
		ui.Icon(c, icons.ChevronRight).FontSize(14).TextColor(tokens.Muted)
	})
	return row.Clicked()
}

// Pages draws the provider list or the chosen provider's page.
func Pages(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	Check(store)
	if nav.Provider == "" {
		list(c, store, tokens, nav)
	} else {
		page(c, store, tokens, nav)
	}
	notice(c, store, tokens)
}

func list(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	messages := store.Copy()
	cl := store.State.Cloud
	kit.Card(c, tokens).Children(func() {
		for i, id := range providers() {
			if i > 0 {
				kit.Divider(c, tokens)
			}
			detail := messages.Text(copy.CloudKeyLabel)
			if id == runtime.CloudChatGPT {
				detail = messages.Text(copy.CloudChatGPTDetail)
			}
			if cl.Phase == runtime.CloudConnected && cl.Provider == id {
				detail = messages.Text(copy.CloudConnected)
			}
			if link(c, tokens, "provider-"+id, config.CloudProviders[id], detail, func() { Mark(c, tokens, id, 20) }) {
				nav.Provider = id
			}
		}
	})
}

// page connects the provider, or, once it is the connection, shows it with
// Disconnect and the model choice.
func page(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	cl := store.State.Cloud
	if cl.Phase == runtime.CloudConnected && cl.Provider == nav.Provider {
		connected(c, store, tokens)
		model(c, store, tokens)
		return
	}
	if nav.Provider == runtime.CloudChatGPT {
		chatgpt(c, store, tokens)
	} else {
		key(c, store, tokens, nav)
	}
}

func chatgpt(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	cl := store.State.Cloud
	kit.Card(c, tokens).Children(func() {
		ui.Column(c).Padding(20, 16).Gap(14).AlignItems(ui.Center).Children(func() {
			Mark(c, tokens, runtime.CloudChatGPT, 32)
			if cl.Phase == runtime.CloudSigningIn {
				ui.Row(c.Key("waiting")).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(14, 14)
					ui.Text(c, messages.Text(copy.CloudSigningIn)).FontSize(theme.Body).TextColor(tokens.Muted)
				})
				cancel(c, store, tokens)
				return
			}
			ui.Text(c, messages.Text(copy.CloudChatGPTDetail)).FontSize(theme.Body).TextColor(tokens.Muted)
			if kit.Button(c.Key("chatgpt-sign-in"), tokens, kit.Primary, messages.Text(copy.CloudChatGPTSignIn), !cl.Busy()).Clicked() {
				store.Actions.SignInChatGPT()
			}
		})
	})
}

// key is the API key field and Connect.
func key(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	messages := store.Copy()
	cl := store.State.Cloud
	kit.Card(c, tokens).Children(func() {
		ui.Row(c).Padding(14, 16).Gap(8).AlignItems(ui.Center).Children(func() {
			Mark(c, tokens, nav.Provider, 20)
			ui.TextInput(c.Key("cloud-key"), &nav.Key).Password().Placeholder(messages.Text(copy.CloudKeyPlaceholder)).
				Label(messages.Text(copy.CloudKeyLabel)).Grow(1).Shrink(1)
			if cl.Phase == runtime.CloudVerifying {
				ui.Spinner(c).Size(14, 14)
				cancel(c, store, tokens)
				return
			}
			if kit.Button(c.Key("cloud-connect"), tokens, kit.Primary, messages.Text(copy.CloudConnect), nav.Key != "" && !cl.Busy()).Clicked() {
				store.Actions.ConnectCloudKey(nav.Provider, nav.Key)
				nav.Key = ""
			}
		})
	})
}

func cancel(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	if kit.Button(c.Key("cloud-cancel"), tokens, kit.Ghost, store.Copy().Text(copy.AccCancel), true).Clicked() {
		store.Actions.CancelCloud()
	}
}

// connected names the provider and the ChatGPT account, with Disconnect.
func connected(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	cl := store.State.Cloud
	detail := cl.Email
	if cl.Provider != runtime.CloudChatGPT {
		detail = messages.Text(copy.CloudKeyLabel)
	}
	kit.Card(c, tokens).Children(func() {
		ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Children(func() {
			Mark(c, tokens, cl.Provider, 20)
			ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
				ui.Text(c, messages.Text(copy.CloudConnected)).FontSize(theme.Body).FontWeight(500)
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted).Selectable()
			})
			if kit.Button(c.Key("cloud-disconnect"), tokens, kit.Outline, messages.Text(copy.CloudDisconnect), true).Clicked() {
				store.Actions.DisconnectCloud()
			}
		})
	})
}

// model chooses among the listed models; typing filters a long list.
func model(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	cl := store.State.Cloud
	chosen := store.State.Preferences.CloudModel
	// Polishing waits on a model; the row says so until one is chosen.
	detail := ""
	if chosen == "" {
		detail = messages.Text(copy.CloudModelPlaceholder)
	}
	kit.Card(c, tokens).Children(func() {
		kit.Row(c, tokens, messages.Text(copy.CloudModelLabel), detail, func() {
			if cl.Listing {
				ui.Spinner(c).Size(14, 14)
			}
			if ui.Combobox(c.Key("cloud-model"), &chosen, cl.Models).Label(messages.Text(copy.CloudModelLabel)).Width(220).Changed() {
				store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.CloudModel = chosen })
			}
		})
	})
}

// failures maps each shown Failure to its copy; a cancel shows nothing.
var failures = map[string]copy.Key{
	runtime.CloudCodeSignIn: copy.CloudFailSignIn, runtime.CloudCodeTimeout: copy.CloudFailTimeout,
	runtime.CloudCodePortInUse: copy.CloudFailPort, runtime.CloudCodeNoSubscription: copy.CloudFailNoPlan,
	runtime.CloudCodeRejected: copy.CloudFailRejected, runtime.CloudCodeNetwork: copy.CloudFailNetwork,
	runtime.CloudCodeStore: copy.CloudFailStore, runtime.CloudCodeModels: copy.CloudFailModels,
}

// FailureKey is the copy for a Failure, or "" when none is shown.
func FailureKey(code string) copy.Key { return failures[code] }

// notice states the last failure; a failed model list offers a retry.
func notice(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	cl := store.State.Cloud
	key := FailureKey(cl.Failure)
	if key == "" {
		return
	}
	messages := store.Copy()
	ui.Row(c.Key("cloud-notice")).Gap(8).Padding(0, 4).AlignItems(ui.Center).Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
		ui.Icon(c, icons.TriangleAlert).FontSize(13).TextColor(tokens.Danger)
		ui.Text(c, messages.Text(key)).FontSize(theme.Caption).Grow(1).Shrink(1)
		if cl.Phase == runtime.CloudConnected && (cl.Failure == runtime.CloudCodeModels || cl.Failure == runtime.CloudCodeNetwork) {
			if kit.Button(c.Key("cloud-retry"), tokens, kit.Outline, messages.Text(copy.Retry), !cl.Listing).Clicked() {
				store.Actions.ListCloudModels()
			}
		}
	})
}
