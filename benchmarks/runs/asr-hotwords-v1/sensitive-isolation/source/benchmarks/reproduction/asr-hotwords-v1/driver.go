// Command driver measures raw production offline ASR with reference-free requests.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"arion.local/core/domain/ports"
	"arion.local/desktop/infrastructure/asr"
	"arion.local/desktop/infrastructure/audio"
)

type request struct {
	ID          string   `json:"id"`
	CaseID      string   `json:"caseId"`
	Purpose     string   `json:"purpose"`
	Arm         string   `json:"arm"`
	AudioPath   string   `json:"audioPath"`
	AudioSHA256 string   `json:"audioSha256"`
	Hotwords    []string `json:"hotwords"`
}

func run() error {
	input := flag.String("input", "", "reference-free request plan")
	model := flag.String("model", "", "verified MLX model directory")
	flag.Parse()
	data, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	var requests []request
	if err = json.Unmarshal(data, &requests); err != nil {
		return err
	}
	start := time.Now()
	engine, err := asr.LoadOffline(asr.Spec{Kind: asr.QwenMLX, Directory: *model, SampleRate: 16000, Threads: 4, EndpointSilence: 0.8})
	if err != nil {
		return err
	}
	defer engine.Close()
	out := json.NewEncoder(os.Stdout)
	if err = out.Encode(map[string]any{"type": "loaded", "elapsedMs": float64(time.Since(start).Microseconds()) / 1000}); err != nil {
		return err
	}
	for _, r := range requests {
		bytes, err := os.ReadFile(r.AudioPath)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(bytes)
		if hex.EncodeToString(hash[:]) != r.AudioSHA256 {
			return fmt.Errorf("changed audio: %s", r.ID)
		}
		samples, err := audio.ReadWAV(r.AudioPath)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		start := time.Now()
		text, err := engine.Finalize(ctx, ports.FinalizeRequest{Samples: samples, SampleRate: 16000, Hotwords: append([]string(nil), r.Hotwords...)})
		elapsed := float64(time.Since(start).Microseconds()) / 1000
		cancel()
		row := map[string]any{"type": "result", "request": r, "text": text, "elapsedMs": elapsed, "samples": len(samples)}
		if err != nil {
			row["error"] = err.Error()
		}
		if encodeErr := out.Encode(row); encodeErr != nil {
			return encodeErr
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
