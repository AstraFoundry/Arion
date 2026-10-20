package dictation

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"testing"
	"time"
)

func TestUncertainPasteCleanupWarningRemainsVisible(t *testing.T) {
	now := time.Now()
	d := runtime.Dictation{State: runtime.StateCompleted, Outcome: runtime.OutcomeUnknown, Failure: runtime.CodeClipboardRestoreFailed, Changed: now}
	if PhaseOf(d, now) != Failed || Remaining(d, now) != FailedVisible {
		t.Fatal("cleanup warning hidden by completion")
	}
	notice := NoticeOf(d, runtime.Permissions{})
	if notice.Title != copy.FailClipboardRestoreTitle || notice.Hint != copy.FailClipboardRestoreHint {
		t.Fatalf("notice: %+v", notice)
	}
	if OutcomeLabel(d, true) != copy.PillPasted {
		t.Fatal("uncertain paste claimed verified insertion")
	}
}
