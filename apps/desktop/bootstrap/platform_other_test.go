//go:build !darwin && !windows

package bootstrap

import (
	"context"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func TestOtherPlatformRefusesInputMethodAndHoldKeys(t *testing.T) {
	p := newPlatform()
	if p.NewInputMethod(nil, nil) != nil {
		t.Fatal("input method")
	}
	if domain.CodeOf(p.StartHoldKeys(nil, nil)) != domain.CodeTargetUnsupported {
		t.Fatal("hold key")
	}
	if text, _, err := p.ReadSelection(); text != "" || domain.CodeOf(err) != domain.CodeTargetUnsupported {
		t.Fatal("selection read")
	}
	if _, _, _, _, ok := p.SelectionAnchor(); ok {
		t.Fatal("selection anchor")
	}
	if domain.CodeOf(p.StartSelectionWatch(func() {})) != domain.CodeTargetUnsupported {
		t.Fatal("selection watch")
	}
	targets := p.NewTargets(nil, TargetGates{})
	if _, err := targets.Capture(context.Background(), contracts.Identity{}); domain.CodeOf(err) != domain.CodeTargetUnsupported {
		t.Fatal(err)
	}
	if targets.FocusOwner() != 0 || targets.Follow("") {
		t.Fatal("target claimed a field")
	}
}
