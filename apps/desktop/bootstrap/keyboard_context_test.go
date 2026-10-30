package bootstrap

import (
	"context"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type contextTargets struct {
	Targets
	reads, captures, discards int
	app                       string
	sensitive                 bool
}

func (t *contextTargets) Capture(_ context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	t.captures++
	return contracts.TargetSnapshot{Identity: id, Target: "target", Application: t.app, Sensitive: t.sensitive}, nil
}
func (t *contextTargets) ReadContext(_ context.Context, id contracts.Identity, token contracts.TargetToken) (contracts.ContextSnapshot, error) {
	t.reads++
	return contracts.ContextSnapshot{Identity: id, Target: token, SafeToRead: true, Before: "beforepreedit", After: "after", Screen: "window"}, nil
}
func (t *contextTargets) Discard(contracts.TargetToken) { t.discards++ }

type contextRankFunc func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error)

func (f contextRankFunc) RankKeyboard(ctx context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	return f(ctx, r)
}
func TestKeyboardContextConsentAndTarget(t *testing.T) {
	for _, mode := range []string{"off", "matching", "foreign", "secure", "denied", "changed-mark"} {
		t.Run(mode, func(t *testing.T) {
			target := &contextTargets{app: "editor", sensitive: mode == "secure"}
			req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "session", Generation: 1}, Application: "editor", Preedit: "preedit", Revision: 1, Input: "word", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "字", Source: "phrase"}, End: 4}}}
			if mode == "foreign" {
				target.app = "other"
			}
			if mode == "denied" {
				req.Application = "com.apple.Passwords"
			}
			if mode == "changed-mark" {
				req.Preedit = "old"
			}
			called := false
			traceReason := ""
			ranker := keyboardContext{trace: func(reason string, err error) {
				if err == nil {
					t.Fatal("failure trace lacks error")
				}
				traceReason = reason
			}, targets: target, cursor: func() bool { return true }, enabled: func() bool { return mode != "off" }, ranker: contextRankFunc(func(_ context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
				called = true
				if mode == "matching" && (r.Before != "before" || r.After != "after" || r.Screen != "window") {
					t.Fatal("context omitted")
				}
				return contracts.KeyboardRankResult{}, nil
			})}
			_, err := ranker.RankKeyboard(context.Background(), req)
			if mode == "matching" || mode == "off" {
				if err != nil || !called {
					t.Fatal(err)
				}
			} else if err == nil || called {
				t.Fatal("unsafe request admitted")
			}
			if mode != "matching" && mode != "changed-mark" && target.reads != 0 {
				t.Fatal("unsafe text read")
			}
			if (mode == "off" || mode == "denied") && target.captures != 0 {
				t.Fatal("unnecessary capture")
			}
			expected := map[string]string{"foreign": "context_target", "secure": "context_target", "denied": "context_target", "changed-mark": "context_marked_mismatch"}[mode]
			if traceReason != expected {
				t.Fatalf("trace %q, want %q", traceReason, expected)
			}
			if target.discards != target.captures {
				t.Fatal("captured target retained")
			}
		})
	}
}
