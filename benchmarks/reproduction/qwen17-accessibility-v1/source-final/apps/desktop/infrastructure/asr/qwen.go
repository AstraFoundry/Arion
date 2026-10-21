package asr

import (
	"arion.local/core/domain"
	"strings"
)

// qwenTranscript separates the model's language envelope from transcription.
// Qwen's parse_asr_output contract also accepts a text-only result. sherpa
// 1.13.8 also requires the decoded token prefix to start exactly with
// "language "; observed 1.7B tokens can start with " language" instead.
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
