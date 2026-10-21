package dictation

import (
	"testing"
	"time"
)

// run advances in 16 ms frames until the text is fully shown and faded.
func run(t *testing.T, w *Typewriter, session, text string, final bool, start time.Time) (time.Time, []Frame) {
	t.Helper()
	now := start
	var frames []Frame
	for i := 0; i < 1000; i++ {
		f := w.Update(session, text, final, now)
		frames = append(frames, f)
		if !f.Animating {
			return now, frames
		}
		now = now.Add(16 * time.Millisecond)
	}
	t.Fatal("typewriter never settled")
	return now, frames
}

func TestTypewriterTypesStreamingTextSmoothly(t *testing.T) {
	var w Typewriter
	start := time.Unix(0, 0)
	end, frames := run(t, &w, "s1", "今天下午三点开会", false, start)
	if frames[len(frames)-1].Text != "今天下午三点开会" {
		t.Fatalf("did not finish: %q", frames[len(frames)-1].Text)
	}
	// Characters arrive a few at a time, never all at once.
	for i := 1; i < len(frames); i++ {
		if grew := len([]rune(frames[i].Text)) - len([]rune(frames[i-1].Text)); grew > 2 {
			t.Fatalf("frame %d jumped %d runes", i, grew)
		}
	}
	if d := end.Sub(start); d < 200*time.Millisecond || d > 900*time.Millisecond {
		t.Fatalf("eight runes took %v", d)
	}
	// New characters fade in and become opaque.
	first := 0
	for frames[first].Text == "" {
		first++
	}
	if len(frames[first].Opacity) == 0 || frames[first].Opacity[len(frames[first].Opacity)-1] >= 1 {
		t.Fatalf("new character not fading: %+v", frames[first])
	}
	if last := frames[len(frames)-1]; len(last.Opacity) != 0 {
		t.Fatalf("settled text still fading: %+v", last)
	}
}

func TestTypewriterSpringCatchesUpWithBacklog(t *testing.T) {
	short, long := Typewriter{}, Typewriter{}
	start := time.Unix(0, 0)
	endShort, _ := run(t, &short, "a", "一二三四五六", false, start)
	endLong, _ := run(t, &long, "b", "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十", false, start)
	perShort := endShort.Sub(start) / 6
	perLong := endLong.Sub(start) / 40
	if perLong >= perShort {
		t.Fatalf("a long backlog did not type faster: %v vs %v per rune", perLong, perShort)
	}
}

func TestTypewriterRevisesInPlaceWithoutRetreating(t *testing.T) {
	var w Typewriter
	now, _ := run(t, &w, "s", "我们明天见面", false, time.Unix(0, 0))
	// The ending changes: the same length shows the new text at once.
	if f := w.Update("s", "我们明天再说", false, now.Add(16*time.Millisecond)); f.Text != "我们明天再说" || len(f.Opacity) != 0 {
		t.Fatalf("in-place revision retyped or faded: %+v", f)
	}
	// A rewrite of early text while the tail is still typing never shrinks
	// the line.
	stream := []string{"今天下午三点", "今天下午三点我们开", "今天下午三点，我们开会", "今天下午 3 点，我们开会讨论", "今天下午3点，我们开会讨论一下"}
	var v Typewriter
	at := time.Unix(10, 0)
	shown := 0
	for _, text := range stream {
		for i := 0; i < 6; i++ {
			f := v.Update("r", text, false, at)
			n := len([]rune(f.Text))
			if n < shown && n < len([]rune(text)) {
				t.Fatalf("line retreated from %d to %d runes on %q", shown, n, text)
			}
			shown = n
			at = at.Add(16 * time.Millisecond)
		}
	}
	// A shorter revision cuts the display to it.
	if f := v.Update("r", "今天", false, at); f.Text != "今天" {
		t.Fatalf("shorter revision not cut: %q", f.Text)
	}
}

func TestTypewriterFinalDrainsFastAndSessionResets(t *testing.T) {
	var w Typewriter
	start := time.Unix(0, 0)
	text := "这是一段比较长的最终识别结果需要尽快显示出来"
	end, _ := run(t, &w, "s", text, true, start)
	if d := end.Sub(start); d > time.Duration(len([]rune(text)))*minInterval+FadeIn+50*time.Millisecond {
		t.Fatalf("final text drained in %v", d)
	}
	if f := w.Update("next", "好", false, end); f.Text == text || len([]rune(f.Text)) > 1 {
		t.Fatalf("new session kept old text: %q", f.Text)
	}
}
