//go:build windows

package windows

import (
	"context"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func TestReadOnlyStandardEditRefusesOnlyThatStyle(t *testing.T) {
	if !readOnlyStandardEdit("Edit", esReadOnly) || !readOnlyStandardEdit("RichEdit50W", esReadOnly|esPassword) {
		t.Fatal("read-only edit or rich edit was treated as writable")
	}
	if readOnlyStandardEdit("Edit", 0) || readOnlyStandardEdit("Edit", esPassword) {
		t.Fatal("writable edit was treated as read-only")
	}
	if readOnlyStandardEdit("Button", esReadOnly) || readOnlyStandardEdit("Chrome_RenderWidgetHostHWND", esReadOnly) || readOnlyStandardEdit("", esReadOnly) {
		t.Fatal("non-edit window was treated as a read-only edit")
	}
}

func TestSameProcessOwnedEditorIsNotArion(t *testing.T) {
	const self uint32 = 42
	if !sameProcessIsArion(self, self, "MyGoWindow") {
		t.Fatal("Arion shell was treated as another application")
	}
	if !sameProcessIsArion(self, self, "MyGoFlyoutShadow") {
		t.Fatal("Arion flyout was treated as another application")
	}
	if sameProcessIsArion(self, self, OwnedEditorClassPrefix+"3") {
		t.Fatal("owned editor fixture was treated as Arion")
	}
	if sameProcessIsArion(self, 99, "Notepad") || sameProcessIsArion(self, 0, "MyGoWindow") {
		t.Fatal("another or unreadable process was treated as Arion")
	}
}

// These cases use invalid native handles and must return before any OS input
// or UI Automation read. Actual editor delivery is tested in e2e.
func TestCommitRejectsStaleIdentityWithoutConsumingTarget(t *testing.T) {
	id := contracts.Identity{SessionID: "current", Generation: 2}
	targets := NewTargets(nil, func() Delivery { return DeliveryAutomatic })
	targets.targets["owned"] = &target{identity: id, secure: true}
	request := contracts.CommitRequest{Identity: contracts.Identity{SessionID: "current", Generation: 1}, Target: "owned", Text: "canary"}
	result, err := targets.Commit(context.Background(), request)
	if err != nil || result.Outcome != contracts.NotApplied || result.Failure != domain.CodeStaleRevision {
		t.Fatalf("stale request: %+v %v", result, err)
	}
	request.Identity = id
	result, err = targets.Commit(context.Background(), request)
	if err != nil || result.Outcome != contracts.NotApplied || result.Failure != domain.CodeSecureFieldBlocked {
		t.Fatalf("owned request after stale request: %+v %v", result, err)
	}
	if targets.targets["owned"] != nil {
		t.Fatal("attempted commit retained its target")
	}
}

func TestCancelledCommitConsumesTargetWithoutPosting(t *testing.T) {
	id := contracts.Identity{SessionID: "cancelled", Generation: 1}
	targets := NewTargets(nil, func() Delivery { return DeliveryAutomatic })
	targets.targets["owned"] = &target{identity: id}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := targets.Commit(ctx, contracts.CommitRequest{Identity: id, Target: "owned", Text: "canary"})
	if err != nil || result.Outcome != contracts.NotApplied || result.Failure != domain.CodeCancelled {
		t.Fatalf("cancelled request: %+v %v", result, err)
	}
	if targets.targets["owned"] != nil {
		t.Fatal("cancelled commit retained a replayable target")
	}
}

func TestContextRejectsUnknownSafetyWithoutReading(t *testing.T) {
	id := contracts.Identity{SessionID: "context", Generation: 1}
	for _, secure := range []bool{false, true} {
		targets := NewTargets(func() bool { t.Fatal("unsafe field evaluated context source"); return true }, func() Delivery { return DeliveryAutomatic })
		targets.targets["owned"] = &target{identity: id, focus: 1, secure: secure}
		snapshot, err := targets.ReadContext(context.Background(), id, "owned")
		if err != nil || snapshot.SafeToRead || snapshot.Screen != "" || snapshot.Before != "" || snapshot.After != "" {
			t.Fatalf("secure=%v: %+v %v", secure, snapshot, err)
		}
	}
}

func TestContextRejectsCancelledAndStaleRequests(t *testing.T) {
	id := contracts.Identity{SessionID: "context", Generation: 2}
	targets := NewTargets(nil, func() Delivery { return DeliveryAutomatic })
	targets.targets["owned"] = &target{identity: id, focus: 1, readable: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if snapshot, err := targets.ReadContext(ctx, id, "owned"); err == nil || snapshot.SafeToRead || snapshot.Screen != "" {
		t.Fatalf("cancelled context: %+v %v", snapshot, err)
	}
	id.Generation--
	if snapshot, err := targets.ReadContext(context.Background(), id, "owned"); err == nil || snapshot.SafeToRead || snapshot.Screen != "" {
		t.Fatalf("stale context: %+v %v", snapshot, err)
	}
}
