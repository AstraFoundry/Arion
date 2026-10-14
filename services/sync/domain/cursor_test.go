package domain

import (
	"testing"
	"time"
)

func TestCursorRoundTrip(t *testing.T) {
	encoded, err := EncodeCursor(42)
	if err != nil {
		t.Fatal(err)
	}
	if encoded == "42" {
		t.Fatal("cursor is the decimal revision")
	}
	got, err := DecodeCursor(encoded)
	if err != nil || got != 42 {
		t.Fatalf("got %d err %v", got, err)
	}
	zero, err := EncodeCursor(0)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := DecodeCursor(zero); err != nil || n != 0 {
		t.Fatalf("zero cursor %d %v", n, err)
	}
	if _, err := DecodeCursor("not-a-cursor"); !IsCode(err, CodeInvalidEnvelope) {
		t.Fatal(err)
	}
}

func TestCursorExpired(t *testing.T) {
	anchor := time.Unix(0, 0).UTC()
	ttl := time.Hour
	if CursorExpired(anchor, anchor.Add(ttl-time.Second), ttl) {
		t.Fatal("inside the window")
	}
	if !CursorExpired(anchor, anchor.Add(ttl), ttl) {
		t.Fatal("at the deadline")
	}
}
