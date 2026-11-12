package asr

import (
	"context"
	"encoding/binary"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLiveSpeechSurvivesHotwords runs the pinned speech.cpp Q4_K worker. It is
// the local check that breath-level audio still reaches the model and that a
// published transcript is not an ordered copy of the hotword list. Protocol
// fixtures cover the echo shapes; this test does not claim hotword accuracy.
func TestLiveSpeechSurvivesHotwords(t *testing.T) {
	if os.Getenv("ARION_ASR_LIVE") != "1" {
		t.Skip("set ARION_ASR_LIVE=1 with ARION_ASR_WORKER and ARION_ASR_MODEL to run the local Qwen3 check")
	}
	worker := os.Getenv("ARION_ASR_WORKER")
	model := os.Getenv("ARION_ASR_MODEL")
	modelFile := os.Getenv("ARION_ASR_MODEL_FILE")
	wav := os.Getenv("ARION_ASR_WAV")
	if worker == "" || model == "" || modelFile == "" || wav == "" {
		t.Fatal("ARION_ASR_WORKER, ARION_ASR_MODEL, ARION_ASR_MODEL_FILE and ARION_ASR_WAV are required")
	}
	e, err := load(context.Background(), Spec{Kind: QwenGGUF, SampleRate: 16000, Threads: 4, Directory: model, Files: map[string]string{"model": modelFile}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(e.close)
	speech, err := readPCM16Wav(wav)
	if err != nil {
		t.Fatal(err)
	}
	terms := []string{"北京", "上海", "广州", "深圳"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	published := func(name string, samples []float32, keepWords bool) {
		t.Helper()
		got, err := e.decode(ctx, samples, terms...)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%s %q", name, got)
		if hotwordEcho(got, terms) || strings.Contains(got, "北京") || strings.Contains(got, "上海") {
			t.Fatalf("%s published hotwords: %q", name, got)
		}
		if keepWords && !strings.Contains(got, "Anyhow") && !strings.Contains(got, "help") && !strings.Contains(got, "job") {
			t.Fatalf("%s dropped the utterance: %q", name, got)
		}
	}
	published("speech", speech, true)
	quiet := make([]float32, len(speech))
	for i, sample := range speech {
		quiet[i] = sample * 0.25
	}
	published("quiet-speech", quiet, true)
	breath := make([]float32, 16000)
	for i := range breath {
		if i%2 == 0 {
			breath[i] = 0.02
		} else {
			breath[i] = -0.02
		}
	}
	published("breath", breath, false)
	published("silence", make([]float32, 16000), false)
}

func readPCM16Wav(path string) ([]float32, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(body) < 44 || string(body[0:4]) != "RIFF" || string(body[8:12]) != "WAVE" {
		return nil, os.ErrInvalid
	}
	offset := 12
	var data []byte
	for offset+8 <= len(body) {
		id := string(body[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(body[offset+4 : offset+8]))
		offset += 8
		if offset+size > len(body) {
			return nil, os.ErrInvalid
		}
		chunk := body[offset : offset+size]
		if id == "data" {
			data = chunk
			break
		}
		offset += size
		if size%2 == 1 {
			offset++
		}
	}
	if len(data) < 2 || len(data)%2 != 0 {
		return nil, os.ErrInvalid
	}
	samples := make([]float32, len(data)/2)
	for i := range samples {
		samples[i] = float32(int16(binary.LittleEndian.Uint16(data[i*2:]))) / 32768
	}
	return samples, nil
}
