//go:build windows

package main

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestRetryPackageStageRename(t *testing.T) {
	linkError := func(code syscall.Errno) error {
		return &os.LinkError{Op: "rename", Old: "stage", New: "final", Err: code}
	}
	tests := []struct {
		name         string
		err          error
		failures     int
		wantAttempts int
		wantPauses   int
		wantError    bool
	}{
		{name: "first success", wantAttempts: 1},
		{name: "access denied then released", err: linkError(5), failures: 2, wantAttempts: 3, wantPauses: 2},
		{name: "sharing then released", err: linkError(32), failures: 1, wantAttempts: 2, wantPauses: 1},
		{name: "permanent access denial bounded", err: linkError(5), failures: 20, wantAttempts: 10, wantPauses: 9, wantError: true},
		{name: "permanent sharing bounded", err: linkError(32), failures: 20, wantAttempts: 10, wantPauses: 9, wantError: true},
		{name: "missing source immediate", err: linkError(2), failures: 20, wantAttempts: 1, wantError: true},
		{name: "missing parent immediate", err: linkError(3), failures: 20, wantAttempts: 1, wantError: true},
		{name: "invalid data outside narrow allowlist", err: linkError(13), failures: 20, wantAttempts: 1, wantError: true},
		{name: "lock violation outside narrow allowlist", err: linkError(33), failures: 20, wantAttempts: 1, wantError: true},
		{name: "unclassified error immediate", err: errors.New("unclassified rename error"), failures: 20, wantAttempts: 1, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts, pauses := 0, 0
			var scheduled time.Duration
			got := retryPackageStageRename(func() error {
				attempts++
				if attempts <= tt.failures {
					return tt.err
				}
				return nil
			}, func(d time.Duration) {
				pauses++
				scheduled += d
				if d != 100*time.Millisecond {
					t.Errorf("pause = %v, want 100ms", d)
				}
			})
			if attempts != tt.wantAttempts || pauses != tt.wantPauses {
				t.Fatalf("attempts/pauses = %d/%d, want %d/%d", attempts, pauses, tt.wantAttempts, tt.wantPauses)
			}
			if scheduled > 900*time.Millisecond {
				t.Fatalf("scheduled wait = %v, exceeds 900ms budget", scheduled)
			}
			if tt.wantError && got != tt.err {
				t.Fatalf("error = %v, want original error identity %v", got, tt.err)
			}
			if !tt.wantError && got != nil {
				t.Fatalf("rename = %v, want success", got)
			}
		})
	}
}
