package dictation

import (
	"testing"

	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
)

func TestIMEFailuresExplainActualState(t *testing.T) {
	for _, tc := range []struct {
		detail      string
		title, hint copy.Key
	}{
		{runtime.DetailInputMethod, copy.FailIMETitle, copy.FailIMEHint},
		{runtime.DetailIMEClient, copy.FailUnsupportedTitle, copy.FailIMEClientHint},
		{runtime.DetailIMEUpdating, copy.FailIMEUpdatingTitle, copy.FailIMEUpdatingHint},
	} {
		got := NoticeOf(runtime.Dictation{Failure: runtime.CodeTargetUnsupported, FailureDetail: tc.detail}, runtime.Permissions{})
		if got.Title != tc.title || got.Hint != tc.hint {
			t.Fatalf("%s: %+v", tc.detail, got)
		}
	}
}
