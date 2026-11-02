package domain

import (
	"strings"
	"unicode"
)

// Punctuation is how a polished proposal is punctuated. It applies only
// while polishing is on. Smart keeps the cleanup prompt unchanged and does
// not edit the model's punctuation afterward.
type Punctuation string

const (
	// PunctuationSmart leaves punctuation to the approved cleanup prompt.
	PunctuationSmart Punctuation = "smart"
	// PunctuationFull asks the model for complete punctuation and keeps it.
	PunctuationFull Punctuation = "full"
	// PunctuationOmit asks for complete punctuation, then drops trailing
	// sentence-final marks before any trailing whitespace.
	PunctuationOmit Punctuation = "omit"
	// PunctuationSpaces asks for complete punctuation, then replaces
	// punctuation characters with a single space.
	PunctuationSpaces Punctuation = "spaces"
)

// Known reports whether p is one of the four punctuation modes.
func (p Punctuation) Known() bool {
	switch p {
	case PunctuationSmart, PunctuationFull, PunctuationOmit, PunctuationSpaces:
		return true
	default:
		return false
	}
}

// sentenceFinalMarks end a sentence for the omit mode. A comma is not one
// of them. FinishScene uses its own set for search queries.
const sentenceFinalMarks = "。！？….!?"

// ApplyPunctuation edits a model proposal for omit and spaces. Smart, full,
// and an empty mode return text unchanged. The model is not told a sentence
// type; omit removes the terminal run of sentence-final marks, preserving
// trailing whitespace and punctuation within the text.
func ApplyPunctuation(mode Punctuation, text string) string {
	switch mode {
	case PunctuationOmit:
		return omitSentenceFinal(text)
	case PunctuationSpaces:
		return spacesForPunctuation(text)
	default:
		return text
	}
}

func omitSentenceFinal(text string) string {
	ending := strings.TrimRightFunc(text, unicode.IsSpace)
	trimmed := strings.TrimRight(ending, sentenceFinalMarks)
	if len(trimmed) == len(ending) {
		return text
	}
	return trimmed + text[len(ending):]
}

func spacesForPunctuation(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		if unicode.IsPunct(r) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return collapseSpaces(b.String())
}

func collapseSpaces(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	space := false
	for _, r := range text {
		if r == ' ' {
			if space {
				continue
			}
			space = true
		} else {
			space = false
		}
		b.WriteRune(r)
	}
	return b.String()
}
