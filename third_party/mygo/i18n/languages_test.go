package i18n

import (
	"slices"
	"testing"
)

func TestLanguages(t *testing.T) {
	t.Cleanup(func() { languages.list = nil })
	if Languages() != nil {
		t.Fatal("languages before any catalog")
	}
	declare("en", []string{"de", "zh-Hans"})
	declare("fr", []string{"de", "ja"})
	if got, want := Languages(), []string{"en", "de", "zh-Hans", "fr", "ja"}; !slices.Equal(got, want) {
		t.Errorf("Languages() = %q, want %q", got, want)
	}
}
