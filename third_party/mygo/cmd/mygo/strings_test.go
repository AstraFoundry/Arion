package main

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestParseStrings(t *testing.T) {
	utf16le := func(s string) []byte {
		b := []byte{0xFF, 0xFE}
		for _, u := range utf16.Encode([]rune(s)) {
			b = append(b, byte(u), byte(u>>8))
		}
		return b
	}
	text := `/* The reasons of Info.plist, in German. */
"NSMicrophoneUsageDescription" = "Mikrofon für \"Notizen\"\nund mehr \U00e4 \UD83C\UDF99";
// A line comment.
NSCameraUsageDescription = "Kamera";
"CFBundleName";
`
	want := map[string]string{
		"NSMicrophoneUsageDescription": "Mikrofon für \"Notizen\"\nund mehr ä 🎙",
		"NSCameraUsageDescription":     "Kamera",
		"CFBundleName":                 "CFBundleName",
	}
	for name, data := range map[string][]byte{
		"UTF-8":     []byte(text),
		"UTF-8 BOM": append([]byte("\xEF\xBB\xBF"), text...),
		"UTF-16":    utf16le(text),
	} {
		got, err := parseStrings(data)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %q, %v", name, got, err)
		}
	}

	xmlText := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>NSCameraUsageDescription</key>
	<string>Kamera &amp; mehr</string>
</dict>
</plist>`
	if got, err := parseStrings([]byte(xmlText)); err != nil || got["NSCameraUsageDescription"] != "Kamera & mehr" {
		t.Errorf("XML: %q, %v", got, err)
	}
	if got, err := parseStrings(utf16le(strings.Replace(xmlText, "UTF-8", "UTF-16", 1))); err != nil || got["NSCameraUsageDescription"] != "Kamera & mehr" {
		t.Errorf("XML in UTF-16: %q, %v", got, err)
	}

	for data, want := range map[string]string{
		`"a" = "b"`:        "line 1: want a ; after",
		"\n\"a\" = \"b":    "line 2: want the end of a string",
		`/* never ends`:    "the end of a comment",
		`"a" = ;`:          "a quoted string",
		"bplist00\x00\x01": "binary property list",
	} {
		if _, err := parseStrings([]byte(data)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: %v, want %q", data, err, want)
		}
	}
}
