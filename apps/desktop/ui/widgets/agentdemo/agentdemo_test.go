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
		{0, func(f frame) bool { return !f.Agent && !f.Copied && f.Pointer == rest && f.Lens.Zoom == 1 }, "opens on the prompt card"},
		{1350, func(f frame) bool { return f.Pointer == copyButton && f.Lens.Zoom > 1.5 }, "zooms in as the pointer reaches Copy"},
		{1450, func(f frame) bool { return f.Copied && f.Press > 0 && f.Ripple >= 0 && f.RippleAt == copyButton }, "the click copies"},
		{2400, func(f frame) bool { return f.Agent && !f.Copied && f.Lens.Zoom == 1 }, "the agent opens, zoomed out"},
		{3500, func(f frame) bool { return f.Focused && f.Pasting && !f.Pasted && f.Pointer == input }, "pastes into the focused input"},
		{4000, func(f frame) bool { return f.Pasted && !f.Pasting && !f.Sent }, "the prompt waits in the input"},
		{4700, func(f frame) bool { return f.Sent && !f.Pasted && !f.Focused && f.Pointer == send }, "sending empties the input"},
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
		if d := abs(f.Pointer.X-previous.Pointer.X) + abs(f.Pointer.Y-previous.Pointer.Y); d > 8 {
			t.Fatalf("pointer jumps %.1f DIP at %d ms", d, ms)
		}
		if d := abs(f.Lens.Zoom - previous.Lens.Zoom); d > 0.03 {
			t.Fatalf("zoom jumps %.3f at %d ms", d, ms)
		}
		previous = f
	}
}

func TestShiftKeepsTheStageCovered(t *testing.T) {
	for _, l := range []lens{{1, center}, {1.5, point{0, 0}}, {1.5, point{Width, Height}}, {1.55, copyButton}, {1.2, issue}} {
		x, y := shift(l)
		// Where the stage's corners land, scaled around its center.
		left := Width/2 - Width/2*l.Zoom + x
		top := Height/2 - Height/2*l.Zoom + y
		right := Width/2 + Width/2*l.Zoom + x
		bottom := Height/2 + Height/2*l.Zoom + y
		if left > 0.01 || top > 0.01 || right < Width-0.01 || bottom < Height-0.01 {
			t.Errorf("%+v uncovers the stage: %.1f %.1f %.1f %.1f", l, left, top, right, bottom)
		}
	}
	if x, y := shift(lens{1.5, center}); x != 0 || y != 0 {
		t.Errorf("centered lens shifts %v %v", x, y)
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
