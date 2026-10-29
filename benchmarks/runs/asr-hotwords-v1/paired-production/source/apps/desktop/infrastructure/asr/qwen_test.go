package asr

import "testing"

func TestQwenTranscriptionEnvelope(t *testing.T) {
	for _, item := range []struct{ raw, want string }{
		{"language Chinese<asr_text>实现区域扩张。", "实现区域扩张。"},
		{"language English<asr_text>He seemed to be cursing people.", "He seemed to be cursing people."},
		{" language Chinese\n<asr_text> 中文 mixed text. ", "中文 mixed text."},
		{"language None<asr_text>", ""},
		{"<asr_text>Text without a language label.", "Text without a language label."},
		{"普通转录。", "普通转录。"},
		{"language models are useful", "language models are useful"},
		{"The literal <asr_text> marker is in this sentence.", "The literal <asr_text> marker is in this sentence."},
		{"language English<asr_text>Keep <asr_text> inside the transcription.", "Keep <asr_text> inside the transcription."},
	} {
		got, err := qwenTranscript(item.raw)
		if err != nil || got != item.want {
			t.Errorf("%q: got %q, %v; want %q", item.raw, got, err, item.want)
		}
	}
	if _, err := qwenTranscript("language <asr_text>body"); err == nil {
		t.Fatal("accepted malformed language metadata")
	}
}
