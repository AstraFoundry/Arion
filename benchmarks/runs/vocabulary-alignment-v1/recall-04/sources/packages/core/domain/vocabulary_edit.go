package domain

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrInvalidVocabularyEdit rejects empty, unsafe or identity replacements.
var ErrInvalidVocabularyEdit = errors.New("replacement needs distinct texts without control characters")

// NormalizeVocabularyEdit validates a complete explicit edit before returning
// one replacement per distinct heard form, preserving the first spelling.
// Case-only corrections remain valid, since written capitalization can matter.
func NormalizeVocabularyEdit(froms []string, to string) ([]Replacement, error) {
	if len(froms) == 0 || !validVocabularyText(to) {
		return nil, ErrInvalidVocabularyEdit
	}
	to = strings.TrimSpace(to)
	var pairs []Replacement
	for _, from := range froms {
		if !validVocabularyText(from) {
			return nil, ErrInvalidVocabularyEdit
		}
		from = strings.TrimSpace(from)
		if from == to {
			return nil, ErrInvalidVocabularyEdit
		}
		duplicate := false
		for _, pair := range pairs {
			if SameVocabularySource(pair.From, from) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			pairs = append(pairs, Replacement{From: from, To: to})
		}
	}
	return pairs, nil
}

func validVocabularyText(text string) bool {
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
