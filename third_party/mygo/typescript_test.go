//go:build !js

package mygo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteTypeScriptBindsNothing writes no client for an app that binds
// no services and declares no events, as one of native UI, which has
// nothing for a frontend to call: generate mode leaves no file behind.
func TestWriteTypeScriptBindsNothing(t *testing.T) {
	ipc.Lock()
	services, events := ipc.services, ipc.events
	ipc.services, ipc.events = nil, nil
	ipc.Unlock()
	t.Cleanup(func() {
		ipc.Lock()
		ipc.services, ipc.events = services, events
		ipc.Unlock()
	})
	// Services of MyGo's own, as plugins bind, are no reason for one.
	ipc.Lock()
	ipc.services = []*service{{name: "internal", internal: true}}
	ipc.Unlock()

	out := filepath.Join(t.TempDir(), "src", "mygo.ts")
	if err := WriteTypeScript(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		b, _ := os.ReadFile(out)
		t.Fatalf("binding nothing wrote %s: %q", out, b)
	}
	if _, err := os.Stat(filepath.Dir(out)); !os.IsNotExist(err) {
		t.Errorf("binding nothing made the directory %s", filepath.Dir(out))
	}
}

// TestDocumentServiceTypeScript generates the client of a document
// service: the content's type and the state. (Without the sources, as in
// a test binary run elsewhere, parameters are named arg0.)
func TestDocumentServiceTypeScript(t *testing.T) {
	ipc.Lock()
	services, events := ipc.services, ipc.events
	ipc.services, ipc.events = nil, nil
	ipc.Unlock()
	t.Cleanup(func() {
		ipc.Lock()
		ipc.services, ipc.events = services, events
		ipc.Unlock()
	})
	type Note struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	ds := &Documents[Note]{}
	if err := bind("Note", ds.Service(), false); err != nil {
		t.Fatal(err)
	}
	src, err := GenerateTypeScript()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"content(): Promise<Note>",
		": Note): Promise<void>",
		"state(): Promise<DocumentState>",
		"save(): Promise<void>",
		"readOnly: boolean",
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("the client lacks %q:\n%s", want, src)
		}
	}
}
