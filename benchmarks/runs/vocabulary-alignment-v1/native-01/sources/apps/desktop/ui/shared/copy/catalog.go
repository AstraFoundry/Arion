package copy

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
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
			i += end + 1
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

// Decibels signs a gain, except 0 dB.
func (c Catalog) Decibels(db int) string {
	value := strconv.Itoa(db)
	if db > 0 {
		value = "+" + value
	}
	return c.format(FormatDecibels, "value", value)
}

func (c Catalog) Centimeters(cm int) string {
	return c.format(FormatCentimeters, "value", strconv.Itoa(cm))
}

// GainStop names a gain and the speaking distance it suits.
func (c Catalog) GainStop(db, cm int) string {
	return c.format(FormatGainStop, "gain", c.Decibels(db), "distance", c.Centimeters(cm))
}

// UpdateAvailable and UpdateTo name a newer release.
func (c Catalog) UpdateAvailable(version string) string {
	return c.format(FormatUpdateAvailable, "version", version)
}

// MoreContributors counts the contributors a stack of avatars leaves out.
func (c Catalog) MoreContributors(n int) string {
	return c.format(FormatMoreContributors, "count", strconv.Itoa(n))
}

// RemoveVocabVariant names the individual spelling removed by its action.
func (c Catalog) RemoveVocabVariant(variant string) string {
	return c.format(FormatRemoveVocabVariant, "variant", variant)
}

func (c Catalog) UpdateTo(version string) string {
	return c.format(FormatUpdateTo, "version", version)
}

// DownloadSize states a model's one-time download size.
func (c Catalog) DownloadSize(n int64) string {
	return c.format(FormatDownloadSize, "size", c.Bytes(n))
}

func (c Catalog) Step(index, total int) string {
	return c.format(FormatStep, "index", strconv.Itoa(index), "total", strconv.Itoa(total))
}

// Count groups digits by thousands, as both locales write them.
func (c Catalog) Count(n int) string {
	digits := strconv.Itoa(max(n, 0))
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}

// Duration splits a span into a figure and its unit, so views can set the
// figure larger: whole seconds under a minute, whole minutes under an
// hour, then hours to one decimal.
func (c Catalog) Duration(ms int64) (value, unit string) {
	switch seconds := max(ms, 0) / 1000; {
	case seconds < 60:
		return strconv.FormatInt(seconds, 10), c.Text(UnitSeconds)
	case seconds < 3600:
		return strconv.FormatInt(seconds/60, 10), c.Text(UnitMinutes)
	default:
		return strconv.FormatFloat(float64(seconds)/3600, 'f', 1, 64), c.Text(UnitHours)
	}
}

// Weekday is a day's short name.
func (c Catalog) Weekday(day time.Weekday) string {
	return c.Text([...]Key{WeekdaySun, WeekdayMon, WeekdayTue, WeekdayWed, WeekdayThu, WeekdayFri, WeekdaySat}[day])
}

// MonthDay names a calendar day without its year.
func (c Catalog) MonthDay(t time.Time) string {
	return c.format(FormatMonthDay, "month", c.Month(t.Month()), "day", strconv.Itoa(t.Day()))
}

// Month is a month's short name.
func (c Catalog) Month(month time.Month) string {
	return c.Text([...]Key{MonthJan, MonthFeb, MonthMar, MonthApr, MonthMay, MonthJun, MonthJul, MonthAug, MonthSep,
		MonthOct, MonthNov, MonthDec}[month-1])
}

// DayWords states a day's dictated words.
func (c Catalog) DayWords(t time.Time, words int) string {
	return c.format(FormatDayWords, "date", c.MonthDay(t), "words", c.Count(words))
}

// UpdateDownloaded distinguishes verified staging from installed resources.
func (c Catalog) UpdateDownloaded(version string) string {
	return c.format(FormatUpdateDownloaded, "version", version)
}

// UpdateReadyVersion describes a completed installation awaiting relaunch.
func (c Catalog) UpdateReadyVersion(version string) string {
	return c.format(FormatUpdateReady, "version", version)
}
