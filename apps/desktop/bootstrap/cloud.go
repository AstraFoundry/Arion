package bootstrap

import (
	"context"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/cloudpolish"
	arion "arion.local/desktop/ui/shared/runtime"
)

// cloudRewriteTimeout bounds the cloud's final rewrite: a round trip to a
// provider takes longer than the local worker.
const cloudRewriteTimeout = 15 * time.Second

// polishRouter is the coordinator's rewriter: the local worker or the
// connected cloud model, as the user chose (ADR 022). The choice is read
// per request, so a change applies to the next dictation.
type polishRouter struct{ a *App }

func (r polishRouter) engine() ports.Rewriter {
	if r.a.preferences().PolishEngine == application.PolishCloud {
		return r.a.cloud
	}
	return r.a.rewriter
}

func (r polishRouter) Rewrite(ctx context.Context, req ports.RewriteRequest) (ports.RewriteResult, error) {
	return r.engine().Rewrite(ctx, req)
}

func (r polishRouter) End(id contracts.Identity) {
	r.a.rewriter.End(id)
	r.a.cloud.End(id)
}

// applyCloud projects the cloud connection into the UI state. Switching the
// active provider restores that provider's model; disconnecting clears it.
func (a *App) applyCloud(s cloudpolish.Status) {
	a.state.Cloud = arion.Cloud{Phase: arion.CloudPhase(s.Phase), Provider: s.Provider, Email: s.Email, Models: s.Models,
		Listing: s.Listing, Failure: string(s.Failure), Saved: s.Saved}
	if s.Phase == cloudpolish.Connected {
		if s.Provider != a.cloudProvider {
			a.cloudProvider = s.Provider
			a.rememberCloudModel(s.Model)
		}
		return
	}
	if s.Phase == cloudpolish.Disconnected && a.cloudProvider != "" {
		a.cloudProvider = ""
		a.rememberCloudModel("")
	}
}

func (a *App) rememberCloudModel(model string) {
	prefs := a.preferences()
	if prefs.CloudModel != model {
		prefs.CloudModel = model
		a.savePreferences(prefs)
	}
	a.cloud.SetModel(model)
}

func (x actions) SignInChatGPT()   { go func() { _ = x.a.cloud.SignInChatGPT() }() }
func (x actions) CancelCloud()     { x.a.cloud.Cancel() }
func (x actions) DisconnectCloud() { go func() { _ = x.a.cloud.Disconnect() }() }
func (x actions) ActivateCloud(provider string) {
	go func() { _ = x.a.cloud.Activate(provider) }()
}
func (x actions) ListCloudModels() {
	// Marked now, so the frames before the service reports do not ask again.
	x.a.state.Cloud.Listing = true
	go x.a.cloud.ListModels()
}
func (x actions) ConnectCloudKey(provider, key string) {
	go func() { _ = x.a.cloud.ConnectKey(provider, key) }()
}
