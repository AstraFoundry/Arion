package ui

import "testing"

func TestSafeToOpen(t *testing.T) {
	for url, want := range map[string]bool{
		"https://example.com/a b": true,
		"HTTP://x":                true,
		"mailto:a@b.c":            true,
		"calc":                    false,
		"file:///etc/passwd":      false,
		`\\host\share\x.exe`:      false,
		"javascript:alert(1)":     false,
		" https://x":              false,
		"https://x\n":             false,
		"https:\\\\x":             false,
		"/notes/1":                false,
		"":                        false,
	} {
		if got := SafeToOpen(url); got != want {
			t.Errorf("SafeToOpen(%q) = %v, want %v", url, got, want)
		}
	}
}
