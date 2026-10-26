package ranking_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/contracts"
	"arion.local/desktop/infrastructure/llm/client"
	"arion.local/desktop/infrastructure/llm/engine"
	"arion.local/desktop/infrastructure/llm/prompt"
	"arion.local/desktop/infrastructure/llm/ranking"
)

type nativeLog struct {
	mu     sync.Mutex
	enc    *json.Encoder
	start  time.Time
	failed error
}

func (l *nativeLog) write(kind string, v any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.failed == nil {
		l.failed = l.enc.Encode(map[string]any{"kind": kind, "atMs": float64(time.Since(l.start).Microseconds()) / 1000, "data": v})
	}
}

type nativeChat struct {
	backend ranking.Completer
	log     *nativeLog
}

func (c nativeChat) Complete(ctx context.Context, r prompt.ChatRequest) (engine.Result, error) {
	started := time.Now()
	out, err := c.backend.Complete(ctx, r)
	failure := ""
	if err != nil {
		failure = err.Error()
	}
	c.log.write("inference", map[string]any{"request": r, "response": out, "error": failure, "wallMs": float64(time.Since(started).Microseconds()) / 1000})
	return out, err
}

// This opt-in integration test schedules frozen native snapshots at real
// monotonic offsets. It observes the Go/model boundary, not rendered IMK UI.
func TestNativeKeyboardSchedule(t *testing.T) {
	input, out := os.Getenv("ARION_RANK_SCHEDULE"), os.Getenv("ARION_RANK_OBSERVATIONS")
	if input == "" || out == "" {
		t.Skip("explicit native schedule and observation path required")
	}
	worker, runtime, model := os.Getenv("ARION_RANK_WORKER"), os.Getenv("ARION_RANK_RUNTIME"), os.Getenv("ARION_RANK_MODEL")
	weight, err := os.Open(model + "/Qwen3.5-2B-Q4_K_M.gguf")
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	_, err = io.Copy(h, weight)
	weight.Close()
	if err != nil || hex.EncodeToString(h.Sum(nil)) != os.Getenv("ARION_RANK_MODEL_SHA256") {
		t.Fatal("weight identity mismatch", err)
	}
	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	var schedule []struct {
		AtMs    int                           `json:"atMs"`
		Request contracts.KeyboardRankRequest `json:"request"`
	}
	if err = json.Unmarshal(data, &schedule); err != nil || len(schedule) == 0 {
		t.Fatal("invalid schedule", err)
	}
	file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c, info, err := client.Start(ctx, client.Config{Worker: worker, Runtime: runtime, Model: model, Threads: 4, MaxLength: 4096, MaxNewTokens: 256}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	log := &nativeLog{enc: json.NewEncoder(file), start: time.Now()}
	log.write("load", info)
	r := &ranking.Ranker{Chat: nativeChat{backend: c, log: log}}
	scheduler, err := application.NewKeyboardRanking(r, 1500*time.Millisecond, 75*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer scheduler.Close()
	var waits sync.WaitGroup
	last := -1
	for _, event := range schedule {
		if event.AtMs < last {
			t.Fatal("nonmonotonic schedule")
		}
		last = event.AtMs
		if _, err = ranking.Prepare(event.Request); err != nil {
			t.Fatal(err)
		}
		delay := time.Duration(event.AtMs)*time.Millisecond - time.Since(log.start)
		if delay > 0 {
			timer := time.NewTimer(delay)
			<-timer.C
		}
		log.write("submit", event)
		answer := scheduler.Submit(event.Request)
		waits.Add(1)
		go func(req contracts.KeyboardRankRequest, ch <-chan application.KeyboardRankOutcome) {
			defer waits.Done()
			out := <-ch
			failure := ""
			if out.Err != nil {
				failure = out.Err.Error()
			}
			log.write("outcome", map[string]any{"request": req, "result": out.Result, "error": failure, "canApply": scheduler.CanApply(out)})
		}(event.Request, answer)
	}
	waits.Wait()
	scheduler.Freeze()
	log.write("complete", map[string]any{"snapshots": len(schedule), "nativeUI": false, "debounceMs": 75, "deadlineMs": 1500})
	if log.failed != nil {
		t.Fatal(log.failed)
	}
}
