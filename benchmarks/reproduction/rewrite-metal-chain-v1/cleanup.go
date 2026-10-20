package main

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	"arion.local/core/domain"
)

type request struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}
type response struct {
	ID        string               `json:"id"`
	Source    string               `json:"source"`
	Cleanup   domain.SpeechCleanup `json:"cleanup"`
	CleanupMs float64              `json:"cleanupMs"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	encoder := json.NewEncoder(os.Stdout)
	for scanner.Scan() {
		var item request
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			panic(err)
		}
		started := time.Now()
		cleanup := domain.CleanSpeech(item.Source)
		elapsed := float64(time.Since(started).Nanoseconds()) / 1e6
		text, err := domain.ApplySpeechSpans(item.Source, cleanup.Spans)
		if err != nil {
			panic(err)
		}
		if text != cleanup.Text {
			panic("Invalid cleanup span replay")
		}
		if err := encoder.Encode(response{ID: item.ID, Source: item.Source, Cleanup: cleanup, CleanupMs: elapsed}); err != nil {
			panic(err)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}
