// Package cloud connects a cloud model for polishing in two pages: the
// list, with the provider in use, its model and Disconnect on top, then
// the subscription and the API key providers; and a provider's page with
// its sign-in or API key. Opening a page never changes the provider in
// use: a saved provider becomes it through its Use button, and a new one
// by connecting it. Secrets never enter the UI state; a pasted key leaves
// the navigation state as soon as it is submitted.
package cloud

import (
	"slices"

	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/icons"
	"arion.local/desktop/ui/shared/kit"
	"arion.local/desktop/ui/shared/runtime"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

// Nav is where the pages are: the list while Provider is empty, else that
// provider's page. The page that shows them keeps it.
type Nav struct {
	Provider string
	// Key is the API key being entered on a provider's page.
	Key string
	// Compact shows either the provider in use or, after Change, the
	// providers, so onboarding's step fits its window.
	Compact bool
	// connecting is the provider this page signed in to or sent a key for:
	// once it is in use, the page returns to the list, where its model is.
	connecting string
	// changing is the provider in use when Change was chosen; another one
	// in use ends the change.
	changing string
}

// Back returns from a provider's page to the list, or from Change to the
// provider in use; false when there is nothing to leave.
func (n *Nav) Back() bool {
	switch {
	case n.Provider != "":
		n.Provider, n.Key, n.connecting = "", "", ""
	case n.changing != "":
		n.changing = ""
	default:
		return false
	}
	return true
}

// Title names the current page.
func Title(store runtime.Store, nav *Nav) string {
	if nav.Provider != "" {
		return config.CloudProviders[nav.Provider]
	}
	return store.Copy().Text(copy.CloudTitle)
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
// provider's mark and model, or that nothing is connected. It reports a
// click.
func Summary(c *ui.Context, store runtime.Store, tokens theme.Tokens) bool {
	messages := store.Copy()
	cl := store.State.Cloud
	value := messages.Text(copy.CloudNotConnected)
	if cl.Phase == runtime.CloudConnected {
		value = config.CloudProviders[cl.Provider]
		if m := store.State.Preferences.CloudModel; m != "" {
			value = m
		}
	}
	kit.Divider(c, tokens)
	return kit.LinkRow(c.Key("cloud-summary"), tokens, messages.Text(copy.CloudTitle), value, func() {
		if cl.Phase == runtime.CloudConnected {
			Mark(c, tokens, cl.Provider, 16)
		}
	})
}

// Pages draws the list or the chosen provider's page.
func Pages(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	Check(store)
	cl := store.State.Cloud
	if nav.Provider != "" && nav.connecting == nav.Provider && cl.Phase == runtime.CloudConnected && cl.Provider == nav.Provider {
		nav.Back()
		nav.changing = ""
	}
	if nav.changing != "" && (cl.Phase != runtime.CloudConnected || cl.Provider != nav.changing) {
		nav.changing = ""
	}
	if nav.Provider == "" {
		list(c, store, tokens, nav)
	} else {
		page(c, store, tokens, nav)
	}
	notice(c, store, tokens)
}

// list shows the provider in use, then the providers by how they connect:
// signing in to a subscription, or an API key.
func list(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	messages := store.Copy()
	cl := store.State.Cloud
	if cl.Phase == runtime.CloudConnected && (!nav.Compact || nav.changing == "") {
		kit.Group(c, tokens, messages.Text(copy.CloudInUse), func() { inUse(c, store, tokens) })
		if nav.Compact {
			if kit.Button(c.Key("cloud-change"), tokens, kit.Ghost, messages.Text(copy.CloudChange), true).Clicked() {
				nav.changing = cl.Provider
			}
			return
		}
	}
	kit.Group(c, tokens, messages.Text(copy.CloudSubscriptions), func() {
		providers(c, store, tokens, nav, []string{runtime.CloudChatGPT})
	})
	kit.Group(c, tokens, messages.Text(copy.CloudAPIKeys), func() {
		providers(c, store, tokens, nav, runtime.CloudKeyProviders)
	})
}

// providers lists ids, each opening its page. The provider in use is
// checked; a saved one offers Use, the only way to switch to it.
func providers(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav, ids []string) {
	messages := store.Copy()
	cl := store.State.Cloud
	kit.Card(c, tokens).Children(func() {
		for i, id := range ids {
			if i > 0 {
				kit.Divider(c, tokens)
			}
			trailing := func() {
				switch {
				case cl.Phase == runtime.CloudConnected && cl.Provider == id:
					ui.Icon(c, icons.Check).FontSize(16).TextColor(tokens.AccentInk).Label(messages.Text(copy.CloudInUse))
				case slices.Contains(cl.Saved, id):
					use(c, store, tokens, id, kit.Outline)
				}
			}
			if kit.LinkRowWith(c.Key("provider-"+id), tokens, config.CloudProviders[id], func() { Mark(c, tokens, id, 18) }, trailing) {
				nav.Provider = id
			}
		}
	})
}

// use makes a saved provider the one in use.
func use(c *ui.Context, store runtime.Store, tokens theme.Tokens, id string, variant kit.Variant) {
	if kit.Button(c.Key("cloud-use-"+id), tokens, variant, store.Copy().Text(copy.CloudUse), !store.State.Cloud.Busy()).Clicked() {
		store.Actions.ActivateCloud(id)
	}
}

// page is the provider in use with its model; a saved provider, with Use;
// or the sign-in or API key that connects a new one.
func page(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
	cl := store.State.Cloud
	switch {
	case cl.Phase == runtime.CloudConnected && cl.Provider == nav.Provider:
		inUse(c, store, tokens)
	case slices.Contains(cl.Saved, nav.Provider) && !cl.Busy():
		saved(c, store, tokens, nav.Provider)
	case nav.Provider == runtime.CloudChatGPT:
		chatgpt(c, store, tokens, nav)
	default:
		key(c, store, tokens, nav)
	}
}

// saved is a provider whose credential is stored but not in use.
func saved(c *ui.Context, store runtime.Store, tokens theme.Tokens, id string) {
	messages := store.Copy()
	kit.Card(c, tokens).Children(func() {
		ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Children(func() {
			Mark(c, tokens, id, 20)
			ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
				ui.Text(c, config.CloudProviders[id]).FontSize(theme.Body).FontWeight(500)
				ui.Text(c, messages.Text(copy.CloudSaved)).FontSize(theme.Caption).TextColor(tokens.Muted)
			})
			use(c, store, tokens, id, kit.Primary)
		})
	})
}

func chatgpt(c *ui.Context, store runtime.Store, tokens theme.Tokens, nav *Nav) {
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
				nav.connecting = runtime.CloudChatGPT
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
				nav.connecting = nav.Provider
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

// inUse is the provider in use: its name and ChatGPT account, Disconnect,
// and the model, which a new connection starts on its fast tier.
func inUse(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	cl := store.State.Cloud
	detail := cl.Email
	if cl.Provider != runtime.CloudChatGPT {
		detail = messages.Text(copy.CloudKeyLabel)
	}
	chosen := store.State.Preferences.CloudModel
	kit.Card(c, tokens).Children(func() {
		ui.Row(c).Gap(12).Padding(12, 16).AlignItems(ui.Center).Children(func() {
			Mark(c, tokens, cl.Provider, 20)
			ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
				ui.Text(c, config.CloudProviders[cl.Provider]).FontSize(theme.Body).FontWeight(500)
				ui.Text(c, detail).FontSize(theme.Caption).TextColor(tokens.Muted).Selectable()
			})
			if kit.Button(c.Key("cloud-disconnect"), tokens, kit.Outline, messages.Text(copy.CloudDisconnect), true).Clicked() {
				store.Actions.DisconnectCloud()
			}
		})
		kit.Divider(c, tokens)
		// Polishing waits on a model; the row says so until one is chosen.
		detail := ""
		if chosen == "" {
			detail = messages.Text(copy.CloudModelPlaceholder)
		}
		kit.Row(c, tokens, messages.Text(copy.CloudModelLabel), detail, func() {
			if cl.Listing {
				ui.Spinner(c).Size(14, 14)
			}
			if ui.Combobox(c.Key("cloud-model"), &chosen, cl.Models).Label(messages.Text(copy.CloudModelLabel)).Width(200).Changed() {
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
