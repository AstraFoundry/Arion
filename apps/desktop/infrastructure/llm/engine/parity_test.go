package engine

import (
	"os"
	"sync/atomic"
	"testing"

	"arion.local/desktop/infrastructure/llm/genai"
	"arion.local/desktop/infrastructure/llm/prompt"
)

// TestReuseParity needs a real runtime and model:
// ARION_LLM_RUNTIME=<genai dir> ARION_LLM_MODEL=<model dir> go test -run Parity
// Greedy decoding after prefix reuse, incremental prefill and rewinding
// must produce exactly what a fresh engine produces for the same request.
func TestReuseParity(t *testing.T) {
	runtimeDir, modelDir := os.Getenv("ARION_LLM_RUNTIME"), os.Getenv("ARION_LLM_MODEL")
	if runtimeDir == "" || modelDir == "" {
		t.Skip("ARION_LLM_RUNTIME and ARION_LLM_MODEL not set")
	}
	if err := genai.Load(runtimeDir); err != nil {
		t.Fatal(err)
	}
	config := Config{ModelDir: modelDir, Threads: 4, MaxLength: 4096, MaxNewTokens: 128}
	var stop atomic.Bool
	final := Request{Session: "a", Scene: "chat", Text: "票价是每张八十块，一共买六张，不对，买七张。"}
	cold := func() string {
		e, _, err := Load(config, prompt.Live())
		if err != nil {
			t.Fatal(err)
		}
		defer e.Close()
		r, err := e.Run(final, &stop)
		if err != nil {
			t.Fatal(err)
		}
		return r.Text
	}()
	e, _, err := Load(config, prompt.Live())
	if err != nil {
		t.Fatal(err)
	}
	defer e.Close()
	// Another session first, then live previews of growing stable text.
	if _, err := e.Run(Request{Session: "z", Scene: "chat", Text: "Can you check the logs?"}, &stop); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"票价是每张八十块，", "票价是每张八十块，一共买六张，"} {
		if _, err := e.Run(Request{Session: "a", Scene: "chat", Text: text, Live: true}, &stop); err != nil {
			t.Fatal(err)
		}
	}
	warm, err := e.Run(final, &stop)
	if err != nil {
		t.Fatal(err)
	}
	if warm.ReusedTokens == 0 {
		t.Fatal("no tokens reused from the live previews")
	}
	if warm.Text != cold {
		t.Fatalf("reuse changed the answer:\nwarm %q\ncold %q", warm.Text, cold)
	}
}
