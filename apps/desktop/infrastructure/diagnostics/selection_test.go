package diagnostics

import (
	"testing"
	"time"

	"arion.local/core/domain"
)

func TestSelectionWatchEventCarriesNoText(t *testing.T) {
	if !(Entry{Event: "selection_watch", Code: domain.CodePermissionDenied, At: time.Now()}).valid() {
		t.Fatal("permission denial was rejected")
	}
	if (Entry{Event: "selection_watch", Outcome: "SecretSelection", At: time.Now()}).valid() {
		t.Fatal("selected text was accepted")
	}
}
