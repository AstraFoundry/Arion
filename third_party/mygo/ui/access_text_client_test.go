package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestAccessibilityEditsCustomTextInputClient(t *testing.T) {
	client := &primitiveClient{text: "A😀bc"}
	disabled := false
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(250, 80).Role(RoleTextField).Label("Custom editor").
			HandleTextInput(client).Disabled(disabled).
			Access(func(n *platform.AccessNode) { n.Value = client.text })
	}, 300, 100)
	defer tt.rt.close()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	field := accessNode(t, tt.h.access, platform.RoleTextField, "Custom editor")
	if field.Actions&platform.ActionSetValue == 0 {
		t.Fatal("custom text client does not advertise setValue")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "你😀"})
	if client.text != "你😀" || tt.h.access.Focus != field.ID {
		t.Fatalf("setValue text %q, focus %d", client.text, tt.h.access.Focus)
	}
	client.selection.Range = TextInputRange{Start: 1, End: 3}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessInsertText, Text: "X"})
	if client.text != "你X" {
		t.Fatalf("insertText did not replace the UTF-16 selection: %q", client.text)
	}
	disabled = true
	tt.Frame()
	tt.send(platform.SurfaceEvent{Kind: platform.AccessAction, ID: field.ID, Action: platform.AccessSetValue, Text: "blocked"})
	if client.text != "你X" {
		t.Fatalf("disabled client was edited: %q", client.text)
	}
}
