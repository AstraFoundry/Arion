package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestCodeOfPreservesFailuresThroughRollbackErrorTrees(t *testing.T) {
	registration := Fail(CodeTargetUnsupported, "input source registration failed")
	for _, tc := range []struct {
		name string
		err  error
		want ErrorCode
	}{
		{"direct", registration, CodeTargetUnsupported},
		{"wrapped", fmt.Errorf("install: %w", registration), CodeTargetUnsupported},
		{"joined cleanup", errors.Join(registration, errors.New("cleanup failed")), CodeTargetUnsupported},
		{"uncoded first branch", errors.Join(errors.New("copy failed"), fmt.Errorf("restore: %w", registration)), CodeTargetUnsupported},
		{"nil cleanup", errors.Join(registration, nil), CodeTargetUnsupported},
		{"nil", nil, ""},
		{"uncoded", errors.New("filesystem error"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CodeOf(tc.err); got != tc.want {
				t.Fatalf("code = %q, want %q", got, tc.want)
			}
		})
	}
}
