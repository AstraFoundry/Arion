package domain

import (
	"strings"
	"unicode"
)

// RewriteContext is text around a dictation that a rewrite may consult to
// resolve plausible recognition errors or continue the surrounding text. It comes from
// other applications and is data, never instructions (009).
type RewriteContext struct {
	// Screen is text recognized in the focused window.
	Screen string
	// Before and After are the field's text around the cursor.
	Before string
	After  string
	// History are the user's recent dictations in the same place, oldest
	// first.
	History []string
}

// Context bounds in Unicode scalar values; history keeps the newest items.
const (
	MaxCursorContext  = 200
	MaxHistoryItems   = 5
	MaxHistoryContext = 200
)

// contextTags are the prompt's context section names; context text cannot
// contain them, or prompt control markers.
var contextTags = []string{"<screen>", "</screen>", "<before_cursor>", "</before_cursor>", "<after_cursor>", "</after_cursor>",
	"<history>", "</history>", "<dictation>", "</dictation>"}

// Bounded returns c sanitized and cut to the context bounds: the end of the
// text before the cursor, the start of the text after it and the newest
// history items. Screen text is preserved in full after sanitization; the
// inference worker rejects requests exceeding its token budget explicitly.
func (c RewriteContext) Bounded() RewriteContext {
	out := RewriteContext{
		Screen: sanitizeContext(c.Screen),
		Before: tail(sanitizeContext(c.Before), MaxCursorContext),
		After:  head(sanitizeContext(c.After), MaxCursorContext),
	}
	for _, h := range c.History {
		if h = head(sanitizeContext(h), MaxHistoryContext); h != "" {
			out.History = append(out.History, h)
		}
	}
	if len(out.History) > MaxHistoryItems {
		out.History = out.History[len(out.History)-MaxHistoryItems:]
	}
	return out
}

// Empty reports whether no context is present.
func (c RewriteContext) Empty() bool {
	return strings.TrimSpace(c.Screen) == "" && strings.TrimSpace(c.Before) == "" && strings.TrimSpace(c.After) == "" && len(c.History) == 0
}

// Evidence is all context text, for validation.
func (c RewriteContext) Evidence() string {
	return strings.Join(append([]string{c.Screen, c.Before, c.After}, c.History...), "\n")
}

func sanitizeContext(s string) string {
	s = strings.ReplaceAll(s, "<|", "")
	s = strings.ReplaceAll(s, "|>", "")
	lower := strings.ToLower(s)
	for _, tag := range contextTags {
		for i := strings.Index(lower, tag); i >= 0; i = strings.Index(lower, tag) {
			s, lower = s[:i]+s[i+len(tag):], lower[:i]+lower[i+len(tag):]
		}
	}
	// Collapse runs of blank lines and spaces that OCR produces.
	lines := strings.Split(s, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if l = strings.Join(strings.Fields(l), " "); l != "" {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

func head(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRightFunc(string(r[:n]), unicode.IsSpace)
}

func tail(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimLeftFunc(string(r[len(r)-n:]), unicode.IsSpace)
}

// contextCopied reports a proposal that adds a long run of content the
// dictation does not support: context leaking into the text instead of
// only correcting its spelling.
func contextCopied(source, candidate string) bool {
	have := map[rune]int{}
	for _, r := range contentRunes(strings.ToLower(source)) {
		have[r]++
	}
	cand := contentRunes(strings.ToLower(candidate))
	novel := 0
	for _, r := range cand {
		if have[r] > 0 {
			have[r]--
		} else {
			novel++
		}
	}
	return novel > 16 && novel*5 > len(cand)
}

// contextSupported counts the proposal's content runes beyond the source
// that the context contains.
func contextSupported(source, context, candidate string) int {
	have := map[rune]int{}
	for _, r := range contentRunes(strings.ToLower(source)) {
		have[r]++
	}
	inContext := map[rune]bool{}
	for _, r := range contentRunes(strings.ToLower(context)) {
		inContext[r] = true
	}
	n := 0
	for _, r := range contentRunes(strings.ToLower(candidate)) {
		if have[r] > 0 {
			have[r]--
		} else if inContext[r] {
			n++
		}
	}
	return n
}

// contextDenied are apps whose fields and windows are never read for
// context, whatever the field: password managers and keychains.
var contextDenied = map[string]bool{
	"com.apple.keychainaccess": true, "com.apple.Passwords": true, "com.1password.1password": true,
	"com.agilebits.onepassword7": true, "com.bitwarden.desktop": true, "org.keepassxc.keepassxc": true,
	"com.lastpass.LastPass": true, "com.dashlane.dashlanephonefinal": true,
}

// ContextDenied reports whether the app's content must never be read.
func ContextDenied(app string) bool { return contextDenied[app] }

// repeatedBefore is the longest start of source, at least two content
// characters long, that repeats the end of the text before the cursor,
// compared without case, spaces and punctuation.
func repeatedBefore(before, source string) string {
	end := contentRunes(strings.ToLower(before))
	src := []rune(source)
	best := ""
	content := 0
	for i, r := range src {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			continue
		}
		content++
		if content < 2 || content > len(end) {
			continue
		}
		prefix := contentRunes(strings.ToLower(string(src[:i+1])))
		if string(end[len(end)-len(prefix):]) == string(prefix) {
			best = string(src[:i+1])
		}
	}
	return best
}
