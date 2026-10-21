// Package cloud connects a cloud model for polishing: signing in with a
// ChatGPT plan or pasting an API key, choosing the model, and
// disconnecting. Secrets never enter the UI state; a pasted key leaves the
// draft as soon as it is submitted.
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

// Draft is the API key being entered; the page keeps it between frames.
type Draft struct {
	Provider string
	Key      string
}

// Shown reports whether the cloud rows apply: the cloud engine is chosen.
func Shown(store runtime.Store) bool {
	return store.State.Preferences.PolishEngine == runtime.PolishCloud
}

// Check lists the connected provider's models once while the rows show.
func Check(store runtime.Store) {
	cl := store.State.Cloud
	if Shown(store) && cl.Phase == runtime.CloudConnected && cl.Models == nil && !cl.Listing && cl.Failure == "" {
		store.Actions.ListCloudModels()
	}
}

// Rows are the connection and model rows inside the polish card, each
// after a divider.
func Rows(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft) {
	if !Shown(store) {
		return
	}
	Check(store)
	cl := store.State.Cloud
	switch cl.Phase {
	case runtime.CloudConnected:
		connected(c, store, tokens)
		model(c, store, tokens)
	case runtime.CloudSigningIn:
		kit.Divider(c, tokens)
		kit.Row(c, tokens, config.CloudProviders[runtime.CloudChatGPT], store.Copy().Text(copy.CloudSigningIn), func() {
			ui.Spinner(c).Size(14, 14)
			cancel(c, store, tokens)
		})
		key(c, store, tokens, draft)
	default:
		chatgpt(c, store, tokens)
		key(c, store, tokens, draft)
	}
	notice(c, store, tokens)
}

func chatgpt(c *ui.Context, store runtime.Store, tokens theme.Tokens) {
	messages := store.Copy()
	kit.Divider(c, tokens)
	kit.Row(c, tokens, config.CloudProviders[runtime.CloudChatGPT], messages.Text(copy.CloudChatGPTDetail), func() {
		if kit.Button(c.Key("chatgpt-sign-in"), tokens, kit.Outline, messages.Text(copy.AccSignIn), !store.State.Cloud.Busy()).Clicked() {
			store.Actions.SignInChatGPT()
		}
	})
}

// key is the API key row: the provider, the key and Connect on one line,
// named by the field's placeholder.
func key(c *ui.Context, store runtime.Store, tokens theme.Tokens, draft *Draft) {
	messages := store.Copy()
	cl := store.State.Cloud
	if !slices.Contains(runtime.CloudKeyProviders, draft.Provider) {
		draft.Provider = runtime.CloudKeyProviders[0]
	}
	names := make([]string, len(runtime.CloudKeyProviders))
	for i, id := range runtime.CloudKeyProviders {
		names[i] = config.CloudProviders[id]
	}
	selected := config.CloudProviders[draft.Provider]
	kit.Divider(c, tokens)
	ui.Row(c).Padding(12, 16).Gap(8).AlignItems(ui.Center).Children(func() {
		if ui.Select(c.Key("cloud-provider"), &selected, names).Label(messages.Text(copy.CloudKeyProvider)).Width(150).Changed() {
			draft.Provider = runtime.CloudKeyProviders[slices.Index(names, selected)]
		}
		ui.TextInput(c.Key("cloud-key"), &draft.Key).Password().Placeholder(messages.Text(copy.CloudKeyPlaceholder)).
			Label(messages.Text(copy.CloudKeyLabel)).Grow(1).Shrink(1)
		if cl.Phase == runtime.CloudVerifying {
			ui.Spinner(c).Size(14, 14)
			cancel(c, store, tokens)
			return
		}
		if kit.Button(c.Key("cloud-connect"), tokens, kit.Outline, messages.Text(copy.CloudConnect), draft.Key != "" && !cl.Busy()).Clicked() {
			store.Actions.ConnectCloudKey(draft.Provider, draft.Key)
			draft.Key = ""
		}
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
	kit.Divider(c, tokens)
	kit.Row(c, tokens, config.CloudProviders[cl.Provider], detail, func() {
		if kit.Button(c.Key("cloud-disconnect"), tokens, kit.Outline, messages.Text(copy.CloudDisconnect), true).Clicked() {
			store.Actions.DisconnectCloud()
		}
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
	kit.Divider(c, tokens)
	kit.Row(c, tokens, messages.Text(copy.CloudModelLabel), detail, func() {
		if cl.Listing {
			ui.Spinner(c).Size(14, 14)
		}
		if ui.Combobox(c.Key("cloud-model"), &chosen, cl.Models).Label(messages.Text(copy.CloudModelLabel)).Width(220).Changed() {
			store.Actions.UpdatePreferences(func(p *runtime.Preferences) { p.CloudModel = chosen })
		}
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
	ui.Row(c.Key("cloud-notice")).Gap(8).Padding(0, 16, 12, 16).AlignItems(ui.Center).Transition(ui.ElementTransition{Enter: &ui.Motion{Y: 4}}).Children(func() {
		ui.Icon(c, icons.TriangleAlert).FontSize(13).TextColor(tokens.Danger)
		ui.Text(c, messages.Text(key)).FontSize(theme.Caption).Grow(1).Shrink(1)
		if cl.Phase == runtime.CloudConnected && (cl.Failure == runtime.CloudCodeModels || cl.Failure == runtime.CloudCodeNetwork) {
			if kit.Button(c.Key("cloud-retry"), tokens, kit.Outline, messages.Text(copy.Retry), !cl.Listing).Clicked() {
				store.Actions.ListCloudModels()
			}
		}
	})
}
