package dictation

import (
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"testing"
	"time"
)

func TestIssue110WindowsUnknownCommitIsLabeledPasted(t *testing.T) {
	pasted := ShortcutCommitPasted(runtime.ModeShortcut, runtime.OutcomeUnknown)
	if !pasted {
		t.Fatal("shortcut paste was not marked pasted")
	}
	now := time.Now()
	d := runtime.Dictation{State: runtime.StateCompleted, Outcome: runtime.OutcomeUnknown, Changed: now}
	if PhaseOf(d, now) != Done {
		t.Fatal("unknown commit was not shown as done")
	}
	label := OutcomeLabel(d, pasted)
	if label != copy.PillPasted {
		t.Fatalf("label %s", label)
	}
	if got := copy.For(copy.SimplifiedChinese).Text(label); got != "已粘贴" {
		t.Fatalf("pill %q", got)
	}
	if ShortcutCommitPasted(runtime.ModeShortcut, runtime.OutcomeCopied) {
		t.Fatal("copied outcome was marked pasted")
	}
	copied := OutcomeLabel(runtime.Dictation{Outcome: runtime.OutcomeCopied}, false)
	if copied != copy.PillCopied || copy.For(copy.SimplifiedChinese).Text(copied) != "已复制" {
		t.Fatal("copied label changed")
	}
}

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
	copied := runtime.Dictation{State: runtime.StateCompleted, Outcome: runtime.OutcomeCopied, Changed: now}
	if OutcomeLabel(copied, true) != copy.PillCopied {
		t.Fatal("clipboard copy claimed a paste")
	}
}
