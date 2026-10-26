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

func TestOldIMEClientIsNotReportedAsMissingSpeechModel(t *testing.T) {
	got := NoticeOf(runtime.Dictation{Failure: runtime.CodeArtifactUnverified, FailureDetail: runtime.DetailIMEClientOutdated}, runtime.Permissions{})
	if got.Title != copy.FailIMEProtocolTitle || got.Hint != copy.FailIMEProtocolHint {
		t.Fatalf("old editor client notice: %+v", got)
	}
	got = NoticeOf(runtime.Dictation{Failure: runtime.CodeArtifactUnverified}, runtime.Permissions{})
	if got.Title != copy.FailModelTitle {
		t.Fatalf("model verification failure was misclassified: %+v", got)
	}
}

func TestIMEVerificationFailuresDoNotNameSpeechModels(t *testing.T) {
	for _, tc := range []struct {
		code   runtime.ErrorCode
		detail string
		title  copy.Key
	}{
		{runtime.CodeArtifactUnverified, runtime.DetailIMEArtifactUnverified, copy.FailIMEArtifactTitle},
		{runtime.CodeContractUnsupported, runtime.DetailIMEProtocolUnsupported, copy.FailIMEProtocolTitle},
	} {
		got := NoticeOf(runtime.Dictation{Failure: tc.code, FailureDetail: tc.detail}, runtime.Permissions{})
		if got.Title != tc.title {
			t.Fatalf("input method error %s: %+v", tc.code, got)
		}
	}
}
