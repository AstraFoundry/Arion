package noop

import (
	"context"

	"arion.local/sync/application"
)

// Gate is the self-host default. It does not read tokens or account tables.
type Gate struct {
	quota int64
}

// New returns a gate for the single implicit tenant.
func New(quota int64) Gate { return Gate{quota: quota} }

// Mode is "noop".
func (Gate) Mode() string { return "noop" }

// Authorize returns the local scope. The authorization header is ignored
// because this deployment has no account check.
func (g Gate) Authorize(context.Context, string) (application.Scope, error) {
	return application.Scope{ID: application.LocalScope, Quota: g.quota}, nil
}
