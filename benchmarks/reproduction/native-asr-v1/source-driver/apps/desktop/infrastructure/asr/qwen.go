package asr

import (
	"arion.local/core/domain"
	"strings"
)

// qwenTranscript separates the model's language envelope from transcription.
// Plain-text output is also accepted by the model protocol.
func qwenTranscript(raw string) (string, error) {
	text := strings.TrimSpace(raw)
	metadata, body, tagged := strings.Cut(text, "<asr_text>")
	if !tagged {
		return text, nil
	}
	if metadata == "" {
		return strings.TrimSpace(body), nil
	}
	if !strings.HasPrefix(metadata, "language ") {
		return text, nil
	}
	if strings.TrimSpace(strings.TrimPrefix(metadata, "language ")) == "" {
		return "", domain.Fail(domain.CodeContractUnsupported, "invalid Qwen ASR language envelope")
	}
	return strings.TrimSpace(body), nil
}
