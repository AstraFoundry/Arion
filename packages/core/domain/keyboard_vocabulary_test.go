package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestKeyboardPinyinTransportValidation(t *testing.T) {
	for _, reading := range []string{"", "xing cha gong fang", "lv", "zh uang"} {
		if err := ValidateKeyboardPinyin("星槎工坊", reading); err != nil {
			t.Fatalf("valid transport form %q: %v", reading, err)
		}
	}
	// Syntactic acceptance does not assert that the engine recognizes a syllable.
	for _, reading := range []string{"Xing", "xing  cha", " xing", "xing ", "xing\ncha", "xing'cha", "lü", "xing1", "abcdefg", strings.Repeat("a ", 64) + "a", string([]byte{0xff})} {
		if err := ValidateKeyboardPinyin("星槎", reading); err == nil {
			t.Fatalf("invalid reading accepted: %q", reading)
		}
	}
	if err := ValidateKeyboardPinyin("bad\nterm", "xing"); err == nil {
		t.Fatal("unsafe term accepted")
	}
}

func TestKeyboardProjectionUsesOnlyExplicitCurrentReadings(t *testing.T) {
	entries := []MemoryEntry{
		{ID: "z", To: "星槎", KeyboardPinyin: "xing cha", Origin: OriginEdit, Uses: 900},
		{ID: "a", To: "星槎", KeyboardPinyin: "xing cha", Origin: OriginUser},
		{ID: "hotword-only", To: "热词", Hotword: true, Origin: OriginUser},
	}
	got, err := KeyboardTerms(entries)
	want := []KeyboardTerm{{ID: "a", Text: "星槎", Reading: "xing cha"}, {ID: "z", Text: "星槎", Reading: "xing cha"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("projection %v %v", got, err)
	}
	entries[0], entries[1] = entries[1], entries[0]
	again, err := KeyboardTerms(entries)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatal("source order changed projection")
	}
	entries[1].KeyboardPinyin = "x\ning"
	if partial, err := KeyboardTerms(entries); err == nil || partial != nil {
		t.Fatal("invalid reading produced partial snapshot")
	}
	if _, err := KeyboardTerms([]MemoryEntry{entries[0], entries[0]}); err == nil {
		t.Fatal("duplicate identity accepted")
	}
}
