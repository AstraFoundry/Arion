package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

// RewritePart is either an editable gap or a fixed spelling in source order.
// Fixed text is rendered locally; the formatter returns only the named gaps.
type RewritePart struct {
	Gap   string `json:"gap,omitempty"`
	Text  string `json:"text"`
	Fixed bool   `json:"fixed,omitempty"`
}

type RewriteLayout struct {
	Source string        `json:"source"`
	Parts  []RewritePart `json:"parts"`
}

// SpellingLayout retains each selector decision, literal quotation and exact
// contextual Latin token. Context itself is never part of the formatter input.
func SpellingLayout(p SpellingPreparation, choices string) (RewriteLayout, error) {
	edited := p.Source
	var err error
	if len(p.Hypotheses) > 0 {
		edited, err = ApplySpellingChoices(choices, p)
		if err != nil {
			return RewriteLayout{}, err
		}
	}
	var selected map[string]string
	if len(p.Hypotheses) > 0 {
		if err := json.Unmarshal([]byte(choices), &selected); err != nil {
			return RewriteLayout{}, err
		}
	}
	spans := spellingQuoted(edited)
	shift := 0
	for _, h := range p.Hypotheses {
		word := h.Source
		if selected[h.ID] == "replace" {
			word = h.Candidates[0]
		}
		spans = append(spans, [2]int{h.Start + shift, h.Start + shift + len(word)})
		shift += len(word) - (h.End - h.Start)
	}
	contextTokens := map[string]bool{}
	for _, value := range p.Context {
		for _, token := range latinTerm.FindAllString(value, -1) {
			contextTokens[strings.ToLower(token)] = true
		}
	}
	for _, loc := range latinTerm.FindAllStringIndex(edited, -1) {
		if contextTokens[strings.ToLower(edited[loc[0]:loc[1]])] {
			spans = append(spans, [2]int{loc[0], loc[1]})
		}
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i][0] != spans[j][0] {
			return spans[i][0] < spans[j][0]
		}
		return spans[i][1] > spans[j][1]
	})
	merged := [][2]int{}
	for _, span := range spans {
		if len(merged) > 0 && span[0] < merged[len(merged)-1][1] {
			merged[len(merged)-1][1] = max(merged[len(merged)-1][1], span[1])
		} else {
			merged = append(merged, span)
		}
	}
	layout := RewriteLayout{Source: edited}
	last := 0
	for i, span := range merged {
		layout.Parts = append(layout.Parts, RewritePart{Gap: fmt.Sprintf("g%d", i), Text: edited[last:span[0]]}, RewritePart{Text: edited[span[0]:span[1]], Fixed: true})
		last = span[1]
	}
	if len(merged) > 0 {
		layout.Parts = append(layout.Parts, RewritePart{Gap: fmt.Sprintf("g%d", len(merged)), Text: edited[last:]})
	}
	return layout, nil
}

// Render checks a complete gap protocol, then inserts fixed source bytes.
// A protocol failure is explicit, never repaired by guessing or source fallback.
func (l RewriteLayout) Render(raw string) (string, error) {
	if !utf8.ValidString(raw) {
		return "", errors.New("invalid layout UTF-8")
	}
	d := json.NewDecoder(strings.NewReader(raw))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return "", errors.New("invalid layout object")
	}
	values := map[string]string{}
	for d.More() {
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return "", errors.New("invalid layout key")
		}
		if _, duplicate := values[key]; duplicate {
			return "", errors.New("duplicate layout gap")
		}
		valueToken, err := d.Token()
		value, ok := valueToken.(string)
		if err != nil || !ok {
			return "", errors.New("invalid layout gap text")
		}
		values[key] = value
	}
	if _, err := d.Token(); err != nil {
		return "", errors.New("incomplete layout object")
	}
	if _, err := d.Token(); err != io.EOF {
		return "", errors.New("trailing layout output")
	}
	var out strings.Builder
	count := 0
	for _, part := range l.Parts {
		if part.Fixed {
			out.WriteString(part.Text)
			continue
		}
		value, ok := values[part.Gap]
		if !ok {
			return "", errors.New("missing layout gap")
		}
		count++
		out.WriteString(value)
	}
	if count != len(values) {
		return "", errors.New("unexpected layout gap")
	}
	body := strings.TrimSpace(out.String())
	for _, part := range l.Parts {
		if part.Fixed && strings.Count(body, part.Text) != strings.Count(l.Source, part.Text) {
			return "", errors.New("repeated fixed spelling in layout gaps")
		}
	}
	return body, nil
}
