package cloudpolish

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// tiers lists, per provider, the model families polishing starts on when
// no model was chosen: the fast, inexpensive tier (GPT Luna, Claude Haiku,
// DeepSeek Flash, Gemini Flash-Lite) that suits short rewrites with
// thinking off. Each tier is a set of substrings a model ID must all
// contain; earlier tiers win. IDs change between releases, so a tier
// matches the provider's listed models rather than naming one.
var tiers = map[string][][]string{
	ChatGPT:    {{"luna"}, {"mini"}},
	OpenAI:     {{"luna"}, {"mini"}},
	Anthropic:  {{"haiku"}},
	Google:     {{"flash-lite"}, {"flash"}},
	DeepSeek:   {{"flash"}, {"deepseek-chat"}},
	OpenRouter: {{"openai/", "luna"}, {"anthropic/", "haiku"}, {"deepseek/", "flash"}, {"google/", "flash-lite"}},
}

// unsuited marks variants that are not general text models, or not stable.
var unsuited = []string{"preview", "exp", "realtime", "audio", "tts", "image", "transcribe", "embed", "search", "vision"}

// DefaultModel picks the model polishing starts on from a provider's listed
// models: the newest model of the provider's first tier that has one, an
// undated alias before its dated snapshots. It returns "" when no tier
// matches, and the user chooses.
func DefaultModel(provider string, models []string) string {
	for _, tier := range tiers[provider] {
		best := ""
		for _, id := range models {
			if matches(strings.ToLower(id), tier) && (best == "" || newer(id, best)) {
				best = id
			}
		}
		if best != "" {
			return best
		}
	}
	return ""
}

func matches(id string, tier []string) bool {
	for _, word := range unsuited {
		if strings.Contains(id, word) {
			return false
		}
	}
	for _, part := range tier {
		if !strings.Contains(id, part) {
			return false
		}
	}
	return true
}

// newer orders by version numbers, then prefers the shorter ID: the alias
// over its dated snapshot.
func newer(a, b string) bool {
	va, vb := version(a), version(b)
	for i := 0; i < len(va) && i < len(vb); i++ {
		if va[i] != vb[i] {
			return va[i] > vb[i]
		}
	}
	if len(va) != len(vb) {
		return len(va) > len(vb)
	}
	return len(a) < len(b)
}

// dated matches a snapshot date: 2026-07-01, or 20251001 and other runs
// of four digits or more.
var dated = regexp.MustCompile(`\d{4}-\d{2}-\d{2}|\d{4,}`)

// version is the ID's numbers in order, without dates, so gpt-6-luna is
// newer than gpt-5.6-luna and claude-haiku-4-5 equals
// claude-haiku-4-5-20251001.
func version(id string) []int {
	var out []int
	for _, field := range strings.FieldsFunc(dated.ReplaceAllString(id, ""), func(r rune) bool { return !unicode.IsDigit(r) }) {
		n, _ := strconv.Atoi(field)
		out = append(out, n)
	}
	return out
}
