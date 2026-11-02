package bootstrap

import (
	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/settings"
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
	a.state.Account.UsageReporting = true
	a.suspendUsageUpload()
	a.recordUsageUpload(now, &usage)
	if a.usageUpload.ledger.Days["2026-10-28"].Words != 10 {
		t.Fatal("signed-out usage recorded")
	}
	if a.state.Account.UsageReporting {
		t.Fatal("Settings still shows reporting after sign-out")
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
	a.state.Account.UsageReporting = true
	a.finishUsageUpload(0, nil, policy.ErrAuthorization, time.Now())
	a.recordUsageUpload(time.Now(), &domain.Usage{Words: 10, SpeechMs: 1000})
	if a.usageUpload.session.UsageConsent || len(a.usageUpload.ledger.Days) != 0 {
		t.Fatal("collection continued after authorization refusal")
	}
	if a.state.Account.UsageReporting {
		t.Fatal("Settings still shows reporting after the platform refused it")
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

// After the platform refused a token without consent, sharing from Arion
// reports with that same token once the platform records the answer.
func TestUsageUploadFollowsANewAnswerOnTheSameToken(t *testing.T) {
	a := &App{}
	a.opts.DataDir = t.TempDir()
	a.state.Account.Phase = arion.AccountSignedIn
	// An unreachable loopback issuer keeps the first flush off the network.
	session := account.Session{Issuer: "http://127.0.0.1:1", Subject: "user-1", AccessToken: "token"}
	a.usageUpload.session = session
	a.usageUpload.deniedToken = "token"
	session.UsageConsent = true
	a.usageUpload.deniedToken = ""
	a.state.Account.UsageSharing = arion.UsageShared
	a.configureUsageUpload(session)
	if !a.usageUpload.session.UsageConsent || !a.state.Account.UsageReporting {
		t.Fatalf("recorded consent did not start reporting: %+v", a.usageUpload.session)
	}
}

// A returning account's token may still carry an earlier acceptance. Signing
// in must not report on that claim: only the platform's answer, the one the
// settings switch shows, starts reporting.
func TestUsageUploadWaitsForThePlatformAnswer(t *testing.T) {
	a := &App{prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
	a.opts.DataDir = t.TempDir()
	a.state.Preferences = application.DefaultPreferences("darwin")
	// An unreachable loopback issuer keeps any flush off the network.
	session := account.Session{Issuer: "http://127.0.0.1:1", Subject: "user-1", AccessToken: "token", UsageConsent: true}
	a.applySession(session, arion.AccountSignedIn)
	if a.state.Account.UsageReporting {
		t.Fatal("a consent claim reported before the platform answered")
	}
	for _, answer := range []arion.UsageSharing{arion.UsageUnknown, arion.UsageUndecided, arion.UsageNotShared} {
		a.state.Account.UsageSharing = answer
		a.configureUsageUpload(session)
		if a.state.Account.UsageReporting {
			t.Fatalf("answer %q reported", answer)
		}
	}
	a.state.Account.UsageSharing = arion.UsageShared
	a.configureUsageUpload(session)
	if !a.state.Account.UsageReporting {
		t.Fatal("a shared answer did not report")
	}
}
