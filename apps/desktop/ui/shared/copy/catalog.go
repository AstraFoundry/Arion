package copy

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Locale is a supported interface language, not an ASR language hint.
type Locale string

const (
	English           Locale = "en"
	SimplifiedChinese Locale = "zh-CN"
)

//go:embed locales/*.json
var localeFiles embed.FS

// Catalog is an immutable translation snapshot safe to share between views.
type Catalog struct {
	messages map[Key]string
}

var catalogs = loadCatalogs()

// For returns the requested catalog. Empty means the English first-run default.
// Unsupported preferences must be normalized at the persistence boundary.
func For(locale Locale) Catalog {
	if locale == "" {
		locale = English
	}
	catalog, ok := catalogs[locale]
	if !ok {
		panic(fmt.Sprintf("unsupported UI locale %q", locale))
	}
	return catalog
}

// Text resolves a typed key; empty keys represent an absent optional message.
// Missing translations are defects and are never silently replaced by English.
func (c Catalog) Text(key Key) string {
	if key == "" {
		return ""
	}
	message, ok := c.messages[key]
	if !ok {
		panic(fmt.Sprintf("missing UI message %q", key))
	}
	return message
}

func loadCatalogs() map[Locale]Catalog {
	out := make(map[Locale]Catalog)
	for _, locale := range []Locale{English, SimplifiedChinese} {
		data, err := localeFiles.ReadFile("locales/" + string(locale) + ".json")
		if err != nil {
			panic(err)
		}
		messages, err := decodeCatalog(string(data))
		if err != nil {
			panic(fmt.Sprintf("invalid UI catalog %s: %v", locale, err))
		}
		out[locale] = Catalog{messages: messages}
	}
	if err := validateTranslation(out[English].messages, out[SimplifiedChinese].messages); err != nil {
		panic(fmt.Sprintf("invalid Chinese UI catalog: %v", err))
	}
	return out
}

// decodeCatalog rejects duplicate keys, empty messages and malformed templates.
func decodeCatalog(data string) (map[Key]string, error) {
	decoder := json.NewDecoder(strings.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("catalog must be an object")
	}
	messages := make(map[Key]string)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key := Key(token.(string))
		if _, exists := messages[key]; exists || key == "" {
			return nil, fmt.Errorf("duplicate or empty key %q", key)
		}
		var message string
		if err := decoder.Decode(&message); err != nil {
			return nil, err
		}
		if strings.TrimSpace(message) == "" {
			return nil, fmt.Errorf("empty message %q", key)
		}
		if _, err := placeholders(message); err != nil {
			return nil, fmt.Errorf("message %s: %w", key, err)
		}
		messages[key] = message
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing catalog data")
	}
	return messages, nil
}

func placeholders(message string) ([]string, error) {
	var names []string
	for i := 0; i < len(message); i++ {
		switch message[i] {
		case '}':
			return nil, fmt.Errorf("unmatched closing brace")
		case '{':
			end := strings.IndexByte(message[i+1:], '}')
			if end < 0 {
				return nil, fmt.Errorf("unclosed placeholder")
			}
			name := message[i+1 : i+1+end]
			if name == "" || strings.IndexFunc(name, func(r rune) bool { return r < 'a' || r > 'z' }) >= 0 {
				return nil, fmt.Errorf("invalid placeholder %q", name)
			}
			names = append(names, name)
			i += end + 2
		}
	}
	sort.Strings(names)
	return names, nil
}

func validateTranslation(base, translated map[Key]string) error {
	if len(base) != len(translated) {
		return fmt.Errorf("message count differs: %d vs %d", len(base), len(translated))
	}
	for key, message := range base {
		translation, ok := translated[key]
		if !ok {
			return fmt.Errorf("missing message %q", key)
		}
		want, err := placeholders(message)
		if err != nil {
			return err
		}
		got, err := placeholders(translation)
		if err != nil {
			return err
		}
		if strings.Join(want, ",") != strings.Join(got, ",") {
			return fmt.Errorf("placeholder mismatch for %q", key)
		}
	}
	return nil
}

func (c Catalog) format(key Key, pairs ...string) string {
	// A single replacement pass prevents braces in user values being interpreted.
	replacements := make([]string, len(pairs))
	for i := 0; i < len(pairs); i += 2 {
		replacements[i], replacements[i+1] = "{"+pairs[i]+"}", pairs[i+1]
	}
	return strings.NewReplacer(replacements...).Replace(c.Text(key))
}

// Bytes preserves the existing binary size calculation in either language.
func (c Catalog) Bytes(n int64) string {
	if n >= 1<<30 {
		return c.format(FormatBytesGB, "value", fmt.Sprintf("%.1f", float64(n)/(1<<30)))
	}
	return c.format(FormatBytesMB, "value", strconv.FormatInt((n+(1<<19))/(1<<20), 10))
}

func (c Catalog) Progress(done, total int64) string {
	return c.format(FormatProgress, "done", c.Bytes(done), "total", c.Bytes(total))
}

func (c Catalog) Percent(rate float64) string {
	return c.format(FormatPercent, "value", fmt.Sprintf("%.1f", rate*100))
}

func (c Catalog) Millis(ms float64) string {
	return c.format(FormatMillis, "value", fmt.Sprintf("%.0f", ms))
}

func (c Catalog) Version(version string) string {
	return c.format(FormatVersion, "version", version)
}

func (c Catalog) Step(index, total int) string {
	return c.format(FormatStep, "index", strconv.Itoa(index), "total", strconv.Itoa(total))
}
