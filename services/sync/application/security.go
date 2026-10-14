package application

import (
	"context"
	"slices"
)

// ReceiveSecurityEvent verifies a SET and applies each known event once per
// jti. Unknown event URIs are accepted so a later type does not stall
// delivery. Duplicate jti values return success without a second effect.
func (s *Service) ReceiveSecurityEvent(ctx context.Context, token string) error {
	if s.gate.Mode() != "oidc" {
		return Fail(CodeNotFound)
	}
	event, err := s.events.Verify(ctx, token)
	if err != nil {
		return err
	}
	if event.JTI == "" || event.Issuer == "" || event.Subject == "" || len(event.Types) == 0 {
		return SET("invalid_request", "event rejected")
	}
	tx, err := s.accounts.BeginAccount(ctx)
	if err != nil {
		s.log.Error("security event transaction", "code", "INTERNAL")
		return err
	}
	defer tx.Rollback(ctx)
	fresh, err := tx.ClaimJTI(ctx, event.JTI, s.now())
	if err != nil {
		return err
	}
	if !fresh {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		s.log.Info("security event duplicate", "jti", event.JTI)
		return nil
	}
	order := []string{
		EventSessionRevoked,
		EventEntitlementsChanged,
		EventAccountEnabled,
		EventAccountDisabled,
		EventAccountPurged,
	}
	for _, kind := range order {
		if !slices.Contains(event.Types, kind) {
			continue
		}
		if err := s.applyEvent(ctx, tx, event, kind); err != nil {
			return err
		}
		s.log.Info("security event", "jti", event.JTI, "type", kind)
	}
	return tx.Commit(ctx)
}

func (s *Service) applyEvent(ctx context.Context, tx AccountTx, event SecurityEvent, kind string) error {
	switch kind {
	case EventAccountPurged:
		return tx.Purge(ctx, event.Issuer, event.Subject)
	case EventAccountDisabled:
		if err := tx.SetFrozen(ctx, event.Issuer, event.Subject, true, s.now()); err != nil {
			return err
		}
		return tx.RevokeSessions(ctx, event.Issuer, event.Subject, event.Issued)
	case EventAccountEnabled:
		return tx.SetFrozen(ctx, event.Issuer, event.Subject, false, s.now())
	case EventSessionRevoked:
		return tx.RevokeSessions(ctx, event.Issuer, event.Subject, event.Issued)
	case EventEntitlementsChanged:
		return nil
	default:
		return nil
	}
}
