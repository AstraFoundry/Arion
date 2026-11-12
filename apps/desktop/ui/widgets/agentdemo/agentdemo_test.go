package agentdemo

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/desktop/ui/shared/config"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/theme"
	"github.com/egoist/mygo/ui"
)

func at(ms int) frame { return frameAt(time.Duration(ms) * time.Millisecond) }

func TestStoryOrder(t *testing.T) {
	steps := []struct {
		ms   int
		want func(frame) bool
		what string
	}{
		{0, func(f frame) bool { return !f.Agent && !f.Copied && f.Cursor.At == rest && f.Lens.Zoom == 1 }, "opens on the prompt card"},
		{1350, func(f frame) bool { return f.Cursor.At == copyButton && f.Lens.Zoom > 1.5 }, "zooms in as the pointer reaches Copy"},
		{1450, func(f frame) bool { return f.Copied && f.Cursor.Press > 0 && f.Cursor.RippleAt == copyButton }, "the click copies"},
		{2400, func(f frame) bool { return f.Agent && !f.Copied && f.Lens.Zoom == 1 }, "the agent opens, zoomed out"},
		{3500, func(f frame) bool { return f.Focused && f.Pasting && !f.Pasted && f.Cursor.At == input }, "pastes into the focused input"},
		{4000, func(f frame) bool { return f.Pasted && !f.Pasting && !f.Sent }, "the prompt waits in the input"},
		{4700, func(f frame) bool { return f.Sent && !f.Pasted && !f.Focused && f.Cursor.At == send }, "sending empties the input"},
		{5500, func(f frame) bool { return f.Reply > 0 && f.Reply < 1 && !f.Issue }, "the reply streams"},
		{7000, func(f frame) bool { return f.Reply == 1 && f.Issue && f.Lens.Focus == issue }, "the issue opens, in focus"},
		{9300, func(f frame) bool { return f.Issue && f.Lens.Zoom == 1 }, "zooms out before looping"},
	}
	for _, step := range steps {
		if f := at(step.ms); !step.want(f) {
			t.Errorf("%d ms: %s: %+v", step.ms, step.what, f)
		}
	}
	if final := frameAt(Final); !final.Issue || final.Reply != 1 {
		t.Errorf("final frame %+v", final)
	}
	if loop := frameAt(Cycle + time.Millisecond); loop.Agent || loop.Issue {
		t.Errorf("the loop starts over: %+v", loop)
	}
}

func TestPointerAndLensMoveContinuously(t *testing.T) {
	previous := at(0)
	for ms := 10; ms < int(Cycle/time.Millisecond); ms += 10 {
		f := at(ms)
		if d := abs(f.Cursor.At.X-previous.Cursor.At.X) + abs(f.Cursor.At.Y-previous.Cursor.At.Y); d > 8 {
			t.Fatalf("pointer jumps %.1f DIP at %d ms", d, ms)
		}
		if d := abs(f.Lens.Zoom - previous.Lens.Zoom); d > 0.03 {
			t.Fatalf("zoom jumps %.3f at %d ms", d, ms)
		}
		previous = f
	}
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestFramesRender(t *testing.T) {
	messages := copy.For(copy.English)
	prompt := messages.FeedbackPrompt("https://arion.astrafoundry.si/feedback.md", "1.0.0", "macOS arm64")
	for _, step := range []struct {
		ms   int
		want string
	}{{1500, messages.Text(copy.Copied)}, {2800, messages.FeedbackDemoAsk(config.Agent)}, {4200, prompt}, {7400, messages.Text(copy.FeedbackDemoOpened)}} {
		f := at(step.ms)
		tt := ui.NewTester(func(c *ui.Context) { render(c, theme.Apply(c), messages, prompt, f) }, Width, Height)
		for range 40 {
			tt.Frame()
		}
		want := step.want
		found := false
		for _, text := range tt.Texts() {
			found = found || text == want
		}
		if !found {
			t.Errorf("%d ms: %q missing from %q", step.ms, want, tt.Texts())
		}
		if dir := os.Getenv("ARION_UI_SNAPSHOTS"); dir != "" {
			var buffer bytes.Buffer
			if err := png.Encode(&buffer, tt.Image()); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("agentdemo-%05d.png", step.ms)), buffer.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
