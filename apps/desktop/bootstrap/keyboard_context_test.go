package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

type contextTargets struct {
	Targets
	reads, captures, discards int
	app                       string
	sensitive                 bool
	selected                  string
	readError                 error
	screen                    string
}

func (t *contextTargets) Capture(_ context.Context, id contracts.Identity) (contracts.TargetSnapshot, error) {
	t.captures++
	return contracts.TargetSnapshot{Identity: id, Target: "target", Application: t.app, Sensitive: t.sensitive}, nil
}
func (t *contextTargets) ReadContext(_ context.Context, id contracts.Identity, token contracts.TargetToken) (contracts.ContextSnapshot, error) {
	t.reads++
	if t.readError != nil {
		return contracts.ContextSnapshot{}, t.readError
	}
	screen := "window"
	if t.screen != "" {
		screen = t.screen
	}
	return contracts.ContextSnapshot{Identity: id, Target: token, SafeToRead: true, Before: "beforepreedit", Selected: t.selected, After: "after", Screen: screen}, nil
}

func TestKeyboardLargeWindowReachesRanker(t *testing.T) {
	for _, cursor := range []bool{true, false} {
		t.Run(fmt.Sprint(cursor), func(t *testing.T) {
			target := &contextTargets{app: "editor", screen: strings.Repeat("visible context ", 1500) + "window tail"}
			called, selected := false, false
			r := keyboardContext{targets: target, enabled: func() bool { return true }, cursor: func() bool { return cursor }, trace: func(reason string, err error) {
				if reason == "context_window_selected" && err == nil {
					selected = true
				} else {
					t.Fatalf("unexpected context trace %s: %v", reason, err)
				}
			}, ranker: contextRankFunc(func(_ context.Context, req contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
				called = true
				wantBefore := "confirmed native context"
				if cursor {
					wantBefore = "before"
				}
				if req.Before != wantBefore || len(req.Screen) != domain.MaxKeyboardScreenContextUnits || !strings.HasSuffix(req.Screen, "window tail") {
					t.Fatal("nearby context selection changed")
				}
				return contracts.KeyboardRankResult{}, nil
			})}
			req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "session", Generation: 1}, Application: "editor", Before: "confirmed native context", Preedit: "preedit", Revision: 1, Input: "word", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "字", Source: "phrase"}, End: 4}}}
			_, err := r.RankKeyboard(context.Background(), req)
			if err != nil || !called || !selected || target.captures != target.discards {
				t.Fatalf("large safe window did not reach model: %v", err)
			}
		})
	}
}
func (t *contextTargets) Discard(contracts.TargetToken) { t.discards++ }

type contextRankFunc func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error)

func (f contextRankFunc) RankKeyboard(ctx context.Context, r contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
	return f(ctx, r)
}
func TestKeyboardContextConsentAndTarget(t *testing.T) {
	for _, mode := range []string{"off", "matching", "foreign", "secure", "denied", "changed-mark", "selected-mark", "foreign-selection"} {
		t.Run(mode, func(t *testing.T) {
			target := &contextTargets{app: "editor", sensitive: mode == "secure"}
			req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "session", Generation: 1}, Application: "editor", Preedit: "preedit", Revision: 1, Input: "word", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "字", Source: "phrase"}, End: 4}}}
			if mode == "selected-mark" {
				target.selected = "preedit"
			}
			if mode == "foreign-selection" {
				target.selected = "other"
			}
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
				if mode == "selected-mark" && r.Before != "beforepreedit" {
					t.Fatal("selected preedit removed twice")
				}
				if mode == "matching" && (r.Before != "before" || r.After != "after" || r.Screen != "window") {
					t.Fatal("context omitted")
				}
				return contracts.KeyboardRankResult{}, nil
			})}
			_, err := ranker.RankKeyboard(context.Background(), req)
			if mode == "matching" || mode == "off" || mode == "selected-mark" {
				if err != nil || !called {
					t.Fatal(err)
				}
			} else if err == nil || called {
				t.Fatal("unsafe request admitted")
			}
			if mode != "matching" && mode != "changed-mark" && mode != "selected-mark" && mode != "foreign-selection" && target.reads != 0 {
				t.Fatal("unsafe text read")
			}
			if (mode == "off" || mode == "denied") && target.captures != 0 {
				t.Fatal("unnecessary capture")
			}
			expected := map[string]string{"foreign": "context_target", "secure": "context_target", "denied": "context_target", "changed-mark": "context_marked_mismatch", "foreign-selection": "context_marked_mismatch"}[mode]
			if traceReason != expected {
				t.Fatalf("trace %q, want %q", traceReason, expected)
			}
			if target.discards != target.captures {
				t.Fatal("captured target retained")
			}
		})
	}
}

func TestKeyboardWindowFailureReasons(t *testing.T) {
	reasons := []string{domain.DetailContextWindowNodes, domain.DetailContextWindowDepth, domain.DetailContextWindowText,
		domain.DetailContextWindowRange, domain.DetailContextWindowFrame, domain.DetailContextWindowEmpty, domain.DetailContextWindowTimeout,
		"private error canary"}
	for _, detail := range reasons {
		t.Run(detail, func(t *testing.T) {
			target := &contextTargets{app: "editor", readError: fmt.Errorf("read failed: %w", domain.Fail(domain.CodeTargetUnsupported, detail))}
			reason := ""
			r := keyboardContext{targets: target, enabled: func() bool { return true }, cursor: func() bool { return false },
				trace: func(s string, _ error) { reason = s },
				ranker: contextRankFunc(func(context.Context, contracts.KeyboardRankRequest) (contracts.KeyboardRankResult, error) {
					t.Fatal("model called after failed read")
					return contracts.KeyboardRankResult{}, nil
				})}
			req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "session", Generation: 1}, Application: "editor", Revision: 1, Input: "word", Candidates: []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "字", Source: "phrase"}, End: 4}}}
			_, err := r.RankKeyboard(context.Background(), req)
			expected := detail
			if detail == "private error canary" {
				expected = "context_read"
			}
			if domain.CodeOf(err) != domain.CodeTargetUnsupported || reason != expected {
				t.Fatalf("reason=%q code=%q", reason, domain.CodeOf(err))
			}
		})
	}
}
