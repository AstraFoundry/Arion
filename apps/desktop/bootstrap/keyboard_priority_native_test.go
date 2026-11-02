package bootstrap

import (
	"context"
	"os"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/keyboardmodel"
	"arion.local/desktop/infrastructure/llm/completion"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/modelruntime"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/models"
)

// Explicit authored worker smoke; this does not drive an installed editor.
func TestLocalKeyboardResumesAfterDictation(t *testing.T) {
	resources, store := os.Getenv("ARION_KEYBOARD_TEST_RESOURCES"), os.Getenv("ARION_KEYBOARD_TEST_STORE")
	if resources == "" || store == "" {
		t.Skip("explicit bundled resources and provisioned model store required")
	}
	catalog, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	fast := keyboardmodel.New(catalog, keyboardmodel.Locate(resources))
	defer fast.Disable()
	settled := rewriting.New(catalog, provision.NewStore(store), rewriting.Locate(resources), 4, modelruntime.Platform())
	defer settled.Disable()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if err = fast.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	if err = settled.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "synthetic", Generation: 1}, Revision: 1, Input: "shijie", Before: "欢迎来到这个", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "c0", Text: "世界", Source: "phrase"}, End: 6},
		{Candidate: domain.Candidate{ID: "c1", Text: "时节", Source: "phrase"}, End: 6}}}
	if _, err = fast.RankKeyboard(ctx, req); err != nil {
		t.Fatal(err)
	}
	for _, terminal := range []contracts.Event{
		{Kind: contracts.CommitResultReady, State: domain.Completed},
		{Kind: contracts.ErrorOccurred, State: domain.Failed},
		{Kind: contracts.SessionStateChanged, State: domain.Cancelled},
	} {
		observeKeyboardSpeechPriority(contracts.Event{Kind: contracts.SessionStateChanged, State: domain.Preparing}, fast, settled)
		if _, err = fast.RankKeyboard(ctx, req); domain.CodeOf(err) != domain.CodeSessionBusy {
			t.Fatalf("fast speech admission: %v", err)
		}
		if _, err = settled.Complete(ctx, prompt.ChatRequest{}); domain.CodeOf(err) != domain.CodeSessionBusy {
			t.Fatalf("settled speech admission: %v", err)
		}
		observeKeyboardSpeechPriority(terminal, fast, settled)
		if _, err = fast.RankKeyboard(ctx, req); err != nil {
			t.Fatalf("fast did not resume after %s: %v", terminal.Kind, err)
		}
		generated := req
		generated.Input = "chonggyxia"
		generated.Preedit = "chong g y xia"
		generated.Before = "我想要把这个项目"
		generated.AllowCompletion = true
		generated.Candidates = []domain.KeyboardCandidate{{Candidate: domain.Candidate{ID: "c0", Text: "崇高一下", Source: "sentence"}, End: len(generated.Input)}}
		started := time.Now()
		out, err := (&completion.Ranker{Chat: settled}).RankKeyboard(ctx, generated)
		if err != nil || out.Completion == nil || out.Completion.Text != "重构一下" {
			t.Fatalf("settled did not resume after %s: %+v %v", terminal.Kind, out, err)
		}
		t.Logf("actual local fast and generated completion resumed after %s/%s in %s", terminal.Kind, terminal.State, time.Since(started))
	}
}
