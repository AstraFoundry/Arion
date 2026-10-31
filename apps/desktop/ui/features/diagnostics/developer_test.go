package diagnostics

import (
	"testing"
	"time"
)

func TestDeveloperUnlockRequiresConsecutiveTaps(t *testing.T) {
	var entry DeveloperEntry
	now := time.Now()
	for i := 0; i < 9; i++ {
		if entry.tap(now.Add(time.Duration(i) * 100 * time.Millisecond)) {
			t.Fatal("unlocked early")
		}
	}
	if entry.tap(now.Add(6 * time.Second)) {
		t.Fatal("expired gesture unlocked")
	}
	for i := 0; i < 8; i++ {
		if entry.tap(now.Add(6 * time.Second)) {
			t.Fatal("unlocked early after expiry")
		}
	}
	if !entry.tap(now.Add(6 * time.Second)) {
		t.Fatal("complete gesture failed")
	}
}
