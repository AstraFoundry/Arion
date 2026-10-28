package gguf

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/desktop/infrastructure/llm/engine"
)

func TestMain(m *testing.M) {
	if marker := os.Getenv("ARION_TEST_UNREADY_GGUF"); marker != "" {
		if err := os.WriteFile(marker, []byte("started"), 0600); err != nil {
			os.Exit(2)
		}
		for {
			time.Sleep(time.Second)
		}
	}
	os.Exit(m.Run())
}

// A real helper process never opens its health socket. Cancelling a subsequent
// rewrite during native reload must stop that process instead of waiting 90s.
func TestCancellationDuringNativeReloadIsBounded(t *testing.T) {
	dir := t.TempDir()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, runtimeExecutable()), data, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "started")
	t.Setenv("ARION_TEST_UNREADY_GGUF", marker)
	var stop atomic.Bool
	observed := make(chan struct{})
	go func() {
		defer close(observed)
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(marker); err == nil {
				stop.Store(true)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		stop.Store(true)
	}()
	e := &Engine{reload: true, runtimeDir: dir, config: engine.Config{ModelDir: t.TempDir(), Threads: 1, MaxLength: 4096, MaxNewTokens: 256}}
	started := time.Now()
	result, err := e.Run(engine.Request{Session: "cancel-during-reload"}, &stop)
	<-observed
	if _, markerErr := os.Stat(marker); markerErr != nil {
		t.Fatal("unready native helper never started")
	}
	if err != nil || !result.Cancelled || time.Since(started) > 3*time.Second {
		t.Fatalf("reload cancellation was not bounded: %+v %v elapsed %v", result, err, time.Since(started))
	}
}
