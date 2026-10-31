package completion

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/llm/spelling"
)

var phoneticClasses = sync.OnceValues(func() (map[string]string, error) {
	dict, err := spelling.CharacterReadings()
	if err != nil {
		return nil, err
	}
	sets := map[string]map[rune]bool{}
	for text, readings := range dict {
		chars := []rune(text)
		if len(chars) != 1 || !unicode.Is(unicode.Han, chars[0]) {
			continue
		}
		for _, raw := range readings {
			syllable := strings.ReplaceAll(strings.TrimRight(raw, "012345"), "ü", "v")
			if len(syllable) == 0 {
				continue
			}
			choices := []string{syllable, syllable[:1]}
			if strings.HasPrefix(syllable, "zh") || strings.HasPrefix(syllable, "ch") || strings.HasPrefix(syllable, "sh") {
				choices = append(choices, syllable[:2])
			}
			for _, key := range choices {
				if sets[key] == nil {
					sets[key] = map[rune]bool{}
				}
				sets[key][chars[0]] = true
			}
		}
	}
	classes := map[string]string{}
	for key, set := range sets {
		chars := make([]rune, 0, len(set))
		for c := range set {
			chars = append(chars, c)
		}
		sort.Slice(chars, func(i, j int) bool { return chars[i] < chars[j] })
		var b strings.Builder
		b.WriteByte('[')
		for i := 0; i < len(chars); i++ {
			j := i
			for j+1 < len(chars) && chars[j+1] == chars[j]+1 {
				j++
			}
			b.WriteRune(chars[i])
			if j > i+1 {
				b.WriteByte('-')
				b.WriteRune(chars[j])
			} else if j == i+1 {
				b.WriteRune(chars[j])
			}
			i = j
		}
		b.WriteByte(']')
		classes[key] = b.String()
	}
	return classes, nil
})

// Grammar follows the engine's displayed syllable boundaries. Expanding a full
// syllable into many one-letter initials would authorize unrelated sentences.
func Grammar(input, preedit string) (string, error) {
	if len(input) < 2 || len(input) > 64 {
		return "", errors.New("completion input outside lattice bounds")
	}
	for _, c := range input {
		if (c < 'a' || c > 'z') && c != '\'' {
			return "", errors.New("invalid completion spelling")
		}
	}
	tokens := strings.Fields(strings.ReplaceAll(preedit, "'", " "))
	if len(tokens) < 2 || len(tokens) > 32 || strings.Join(tokens, "") != strings.ReplaceAll(input, "'", "") {
		return "", errors.New("invalid completion segmentation")
	}
	classes, err := phoneticClasses()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("root ::= \"NONE\" |")
	for i := range tokens {
		fmt.Fprintf(&b, " s%d", i)
	}
	b.WriteByte('\n')
	for i, token := range tokens {
		class, ok := classes[token]
		if !ok {
			return "", errors.New("unknown completion syllable")
		}
		fmt.Fprintf(&b, "s%d ::= %s\n", i, class)
	}
	if b.Len() > 256*1024 {
		return "", errors.New("phonetic lattice exceeds bounds")
	}
	return b.String(), nil
}

// SegmentedReading rechecks the same boundaries after decoding, including when
// an incompatible worker ignores grammar. It never accepts a model reading.
func SegmentedReading(input, preedit, text string) string {
	tokens := strings.Fields(strings.ReplaceAll(preedit, "'", " "))
	chars := []rune(text)
	if len(tokens) != len(chars) || len(tokens) < 2 || len(tokens) > 32 || strings.Join(tokens, "") != strings.ReplaceAll(input, "'", "") {
		return ""
	}
	dict, err := spelling.CharacterReadings()
	if err != nil {
		return ""
	}
	readings := make([]string, len(tokens))
	for i, token := range tokens {
		for _, raw := range dict[string(chars[i])] {
			syllable := strings.ReplaceAll(strings.TrimRight(raw, "012345"), "ü", "v")
			if len(syllable) == 0 {
				continue
			}
			if token == syllable || token == syllable[:1] || ((strings.HasPrefix(syllable, "zh") || strings.HasPrefix(syllable, "ch") || strings.HasPrefix(syllable, "sh")) && token == syllable[:2]) {
				readings[i] = syllable
				break
			}
		}
		if readings[i] == "" {
			return ""
		}
	}
	reading := strings.Join(readings, " ")
	if !domain.ValidKeyboardCompletion(domain.KeyboardCompletion{Input: input, Text: text, Reading: reading}, input) {
		return ""
	}
	return reading
}
