//go:build darwin

package macos

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type pasteFixture struct {
	check func() domain.ErrorCode
	begin func(string) pasteAttempt
}

func (f pasteFixture) validate() domain.ErrorCode {
	if f.check != nil {
		return f.check()
	}
	return ""
}
func (f pasteFixture) prepare(s string) pasteAttempt { return f.begin(s) }

type fakePasteAttempt struct {
	onStage, onPost         func() domain.ErrorCode
	onFinish                func() bool
	stages, posts, finishes int
}

func (f *fakePasteAttempt) stage() domain.ErrorCode {
	f.stages++
	if f.onStage != nil {
		return f.onStage()
	}
	return ""
}
func (f *fakePasteAttempt) post() domain.ErrorCode {
	f.posts++
	if f.onPost != nil {
		return f.onPost()
	}
	return ""
}
func (f *fakePasteAttempt) finish() bool {
	f.finishes++
	if f.onFinish != nil {
		return f.onFinish()
	}
	return true
}

func TestPasteTransactionBoundaries(t *testing.T) {
	for _, step := range []string{"before-prepare", "during-prepare", "after-stage", "focus-after-stage", "secure-after-stage", "stage-fails", "post-rejects", "posted-unknown", "restore-fails"} {
		t.Run(step, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tx := &fakePasteAttempt{}
			code := domain.ErrorCode("")
			fixture := pasteFixture{check: func() domain.ErrorCode { return code }, begin: func(s string) pasteAttempt {
				if s != "中🙂\nline" {
					t.Fatalf("text altered: %q", s)
				}
				if step == "during-prepare" {
					cancel()
				}
				return tx
			}}
			if step == "before-prepare" {
				cancel()
			}
			tx.onStage = func() domain.ErrorCode {
				switch step {
				case "after-stage":
					cancel()
				case "focus-after-stage":
					code = domain.CodeTargetChanged
				case "secure-after-stage":
					code = domain.CodeSecureFieldBlocked
				case "stage-fails":
					return domain.CodeClipboardUnavailable
				}
				return ""
			}
			tx.onPost = func() domain.ErrorCode {
				if step == "post-rejects" {
					return domain.CodeTargetChanged
				}
				return ""
			}
			tx.onFinish = func() bool { return step != "restore-fails" }
			result, err := commitPaste(ctx, contracts.Identity{}, "中🙂\nline", 0, fixture)
			if err != nil {
				t.Fatal(err)
			}
			posted := step == "posted-unknown" || step == "restore-fails"
			if posted {
				if result.Outcome != contracts.Unknown || tx.posts != 1 {
					t.Fatalf("posted result %+v, attempts=%d", result, tx.posts)
				}
			} else if result.Outcome != contracts.NotApplied || result.Failure == "" {
				t.Fatalf("rejected result %+v", result)
			}
			if step != "before-prepare" && tx.finishes != 1 {
				t.Fatalf("cleanup count %d", tx.finishes)
			}
			if step != "post-rejects" && !posted && tx.posts != 0 {
				t.Fatalf("unexpected post %d", tx.posts)
			}
			if step == "restore-fails" && result.Failure != domain.CodeClipboardRestoreFailed {
				t.Fatalf("restore failure lost: %+v", result)
			}
		})
	}
}

func TestPasteRejectsUnrepresentableTextBeforeClipboardAccess(t *testing.T) {
	fixture := pasteFixture{begin: func(string) pasteAttempt { t.Fatal("clipboard accessed"); return nil }}
	for _, text := range []string{"", "before\x00after", string([]byte{0xff})} {
		r, _ := commitPaste(context.Background(), contracts.Identity{}, text, 0, fixture)
		if r.Outcome != contracts.NotApplied || r.Failure == "" {
			t.Fatalf("accepted %q: %+v", text, r)
		}
	}
}

func TestPasteQueueCancellationAndSerialization(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	finished := make(chan struct{})
	first := pasteFixture{begin: func(string) pasteAttempt {
		return &fakePasteAttempt{onPost: func() domain.ErrorCode { close(entered); <-release; return "" }}
	}}
	go func() {
		defer close(finished)
		_, _ = commitPaste(context.Background(), contracts.Identity{}, "first", 0, first)
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	second := pasteFixture{begin: func(string) pasteAttempt { t.Error("cancelled waiter touched clipboard"); return nil }}
	waited := make(chan contracts.CommitResult, 1)
	go func() { r, _ := commitPaste(ctx, contracts.Identity{}, "second", 0, second); waited <- r }()
	cancel()
	select {
	case r := <-waited:
		if r.Failure != domain.CodeCancelled {
			t.Errorf("waiter result %+v", r)
		}
	case <-time.After(time.Second):
		t.Error("cancelled waiter remained blocked")
	}
	close(release)
	<-finished

	var active atomic.Int32
	var wg sync.WaitGroup
	fixture := pasteFixture{begin: func(string) pasteAttempt {
		if active.Add(1) != 1 {
			t.Error("overlapping clipboard owners")
		}
		return &fakePasteAttempt{onFinish: func() bool { active.Add(-1); return true }}
	}}
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _ := commitPaste(context.Background(), contracts.Identity{}, "text", time.Millisecond, fixture)
			if r.Outcome != contracts.Unknown {
				t.Errorf("result %+v", r)
			}
		}()
	}
	wg.Wait()
	if active.Load() != 0 {
		t.Fatal("clipboard ownership leaked")
	}
}

func TestCapturedIdentityAndDuplicateCommit(t *testing.T) {
	targets := NewTargets()
	id := contracts.Identity{SessionID: "owner", Generation: 2}
	targets.targets["captured"] = &target{identity: id, secure: true, route: RouteClipboard}
	r, _ := targets.Commit(context.Background(), contracts.CommitRequest{Identity: contracts.Identity{SessionID: "stale", Generation: 1}, Target: "captured", Text: "wrong"})
	if r.Failure != domain.CodeStaleRevision {
		t.Fatalf("stale result %+v", r)
	}
	r, _ = targets.Commit(context.Background(), contracts.CommitRequest{Identity: id, Target: "captured", Text: "blocked"})
	if r.Failure != domain.CodeSecureFieldBlocked {
		t.Fatalf("secure result %+v", r)
	}
	r, _ = targets.Commit(context.Background(), contracts.CommitRequest{Identity: id, Target: "captured", Text: "again"})
	if r.Failure != domain.CodeTargetChanged {
		t.Fatalf("duplicate result %+v", r)
	}
}

func TestPastePreparationFailureDoesNotPost(t *testing.T) {
	result, err := commitPaste(context.Background(), contracts.Identity{}, "text", 0, pasteFixture{begin: func(string) pasteAttempt { return nil }})
	if err != nil || result.Outcome != contracts.NotApplied || result.Failure != domain.CodeClipboardUnavailable {
		t.Fatalf("preparation result: %+v %v", result, err)
	}
}

func TestCancellationAfterPostPreservesHoldAndCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const hold = 20 * time.Millisecond
	var posted time.Time
	tx := &fakePasteAttempt{onPost: func() domain.ErrorCode { posted = time.Now(); cancel(); return "" }}
	tx.onFinish = func() bool {
		if time.Since(posted) < hold {
			t.Error("cancellation shortened clipboard retention")
		}
		return true
	}
	result, err := commitPaste(ctx, contracts.Identity{}, "text", hold, pasteFixture{begin: func(string) pasteAttempt { return tx }})
	if err != nil || result.Outcome != contracts.Unknown || result.Failure != "" || tx.posts != 1 || tx.finishes != 1 {
		t.Fatalf("post-cancellation result: %+v, transaction %+v", result, tx)
	}
}
