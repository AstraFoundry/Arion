package bootstrap

import (
	"arion.local/core/domain"
	infra "arion.local/desktop/infrastructure/usageupload"
	"arion.local/desktop/internal/account"
	policy "arion.local/desktop/internal/usageupload"
	arion "arion.local/desktop/ui/shared/runtime"
	"path/filepath"
	"testing"
	"time"
)

func TestUsageUploadRequiresConsentAndStopsAtSignOut(t *testing.T) {
	a := &App{}
	a.state.Account.Phase = arion.AccountSignedIn
	a.usageUpload.store = infra.Store{Path: filepath.Join(t.TempDir(), "usage.json")}
	a.usageUpload.ledger = policy.Ledger{Version: 1, Source: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Days: map[string]policy.Day{}, Acked: map[string]policy.Day{}}
	now := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	usage := domain.Usage{Words: 10, SpeechMs: 1000}
	a.recordUsageUpload(now, &usage)
	if len(a.usageUpload.ledger.Days) != 0 {
		t.Fatal("unconsented usage recorded")
	}
	a.usageUpload.session = account.Session{UsageConsent: true}
	a.recordUsageUpload(now, &usage)
	if a.usageUpload.ledger.Days["2026-10-28"].Words != 10 {
		t.Fatal("consented usage missing")
	}
	a.suspendUsageUpload()
	a.recordUsageUpload(now, &usage)
	if a.usageUpload.ledger.Days["2026-10-28"].Words != 10 {
		t.Fatal("signed-out usage recorded")
	}
}

func TestUsageAcknowledgementCannotEraseConcurrentCommitsOrCrossAccounts(t *testing.T) {
	a := &App{}
	a.usageUpload.store = infra.Store{Path: filepath.Join(t.TempDir(), "usage.json")}
	day := policy.Day{Date: "2026-10-28", Visited: true, Sessions: 2, Words: 20}
	a.usageUpload.ledger = policy.Ledger{Source: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Days: map[string]policy.Day{day.Date: day}, Acked: map[string]policy.Day{}}
	a.usageUpload.epoch = 2
	sent := day
	sent.Sessions = 1
	sent.Words = 10
	a.finishUsageUpload(1, []policy.Day{sent}, nil, time.Now())
	if len(a.usageUpload.ledger.Acked) != 0 {
		t.Fatal("stale account acknowledged")
	}
	a.finishUsageUpload(2, []policy.Day{sent}, nil, time.Now())
	if len(a.usageUpload.ledger.Pending()) != 1 {
		t.Fatal("newer counters erased by old acknowledgement")
	}
}

func TestAuthorizationFailureStopsCollectionUntilNewConsentToken(t *testing.T) {
	a := &App{}
	a.state.Account.Phase = arion.AccountSignedIn
	a.usageUpload.store = infra.Store{Path: filepath.Join(t.TempDir(), "usage.json")}
	a.usageUpload.session = account.Session{UsageConsent: true, AccessToken: "old-token"}
	a.usageUpload.ledger = policy.Ledger{Source: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Days: map[string]policy.Day{}, Acked: map[string]policy.Day{}}
	a.finishUsageUpload(0, nil, policy.ErrAuthorization, time.Now())
	a.recordUsageUpload(time.Now(), &domain.Usage{Words: 10, SpeechMs: 1000})
	if a.usageUpload.session.UsageConsent || len(a.usageUpload.ledger.Days) != 0 {
		t.Fatal("collection continued after authorization refusal")
	}
}

func TestUsagePayloadFailureKeepsSnapshotPending(t *testing.T) {
	a := &App{}
	a.usageUpload.store = infra.Store{Path: filepath.Join(t.TempDir(), "usage.json")}
	a.usageUpload.session = account.Session{UsageConsent: true, AccessToken: "token"}
	day := policy.Day{Date: "2026-10-28", Visited: true, Sessions: 1, SpeechMs: 60000, Words: 120}
	a.usageUpload.ledger = policy.Ledger{Source: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Days: map[string]policy.Day{day.Date: day}, Acked: map[string]policy.Day{}}
	now := time.Now()
	a.finishUsageUpload(0, nil, policy.ErrPayload, now)
	if len(a.usageUpload.ledger.Pending()) != 1 || !a.usageUpload.session.UsageConsent || !a.usageUpload.next.After(now) {
		t.Fatal("invalid success response discarded usage or disabled collection")
	}
}
