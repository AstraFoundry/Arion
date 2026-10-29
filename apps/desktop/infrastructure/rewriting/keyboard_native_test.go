package rewriting

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/llm/ranking"
)

// Explicit synthetic inference only; no personal profile or installed IME access.
func TestLocalKeyboardWorker(t *testing.T) {
	bundled, model := os.Getenv("ARION_KEYBOARD_TEST_RUNTIME"), os.Getenv("ARION_KEYBOARD_TEST_MODEL")
	if bundled == "" || model == "" {
		t.Skip("explicit local runtime and model required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, _, err := client.Start(ctx, client.Config{Worker: bundled + "/" + workerName(), Runtime: bundled, Model: model, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	s := &Service{client: c}
	scheduler, err := application.NewKeyboardRanking(&ranking.Ranker{Chat: s}, 750*time.Millisecond, 75*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer scheduler.Close()
	req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "synthetic", Generation: 1}, Revision: 1, Input: "xian", Before: "这条", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "c0", Text: "西安", Source: "phrase"}, End: 4},
		{Candidate: domain.Candidate{ID: "c1", Text: "线", Source: "phrase"}, End: 4}}}
	started := time.Now()
	outcome := <-scheduler.Submit(req)
	t.Logf("production deadline result: elapsed=%s error=%v result=%+v", time.Since(started), outcome.Err, outcome.Result)
	if outcome.Err != nil {
		t.Fatal(outcome.Err)
	}
	if !scheduler.CanApply(outcome) {
		t.Fatal("real inference result was not applicable")
	}
	s.SuspendKeyboard(true)
	req.Revision++
	denied := <-scheduler.Submit(req)
	if domain.CodeOf(denied.Err) != domain.CodeModelMissing {
		t.Fatalf("speech did not deny keyboard work: %v", denied.Err)
	}
}
