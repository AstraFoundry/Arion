//go:build !js

package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// A page muted before its web view is made (WebView2 makes it
// asynchronously) starts muted, stays muted as it navigates, and plays
// again once unmuted. A page that plays nothing is not audible. CI's
// machines may have no audio device, so whether a page that plays is
// audible is TestPageAudible's, run by hand.
func TestPageAudioMuted(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "Audio muted", Width: 300, Height: 200})
	p := w.Page()
	p.SetAudioMuted(true)
	if !p.IsAudioMuted() {
		t.Fatal("IsAudioMuted = false right after SetAudioMuted(true)")
	}
	p.LoadHTML(`<p>one</p>`, "")
	waitFor(t, w, `document.body.textContent === "one" && document.readyState === "complete"`)
	if !p.IsAudioMuted() {
		t.Error("IsAudioMuted = false once the page loaded, muted before")
	}
	if p.IsAudible() {
		t.Error("IsAudible = true for a page playing nothing")
	}
	p.LoadHTML(`<p>two</p>`, "")
	waitFor(t, w, `document.body.textContent === "two" && document.readyState === "complete"`)
	if !p.IsAudioMuted() {
		t.Error("IsAudioMuted = false after a navigation")
	}
	// WebView2 may take a moment to settle a change that follows another
	// at once (CI's Windows read false right after muting again), so
	// these wait for it, and fail only if it never comes.
	p.SetAudioMuted(false)
	wait.Until(t, "IsAudioMuted false after SetAudioMuted(false)", func() bool { return !p.IsAudioMuted() })
	p.SetAudioMuted(true)
	wait.Until(t, "IsAudioMuted true after SetAudioMuted(true) again", p.IsAudioMuted)

	// A burst of changes ends in the last one: WebView2 may lose or
	// reorder IsMuted puts that follow each other at once.
	for range 5 {
		p.SetAudioMuted(false)
		p.SetAudioMuted(true)
	}
	wait.Until(t, "IsAudioMuted true after a burst ending muted", p.IsAudioMuted)
	mustEval(t, w, `1`) // a round trip, for a late put to land
	time.Sleep(200 * time.Millisecond)
	if !p.IsAudioMuted() {
		t.Error("IsAudioMuted = false a while after a burst ending muted")
	}
	p.SetAudioMuted(false)
	wait.Until(t, "IsAudioMuted false after unmuting a burst", func() bool { return !p.IsAudioMuted() })
	mustEval(t, w, `1`)
	time.Sleep(200 * time.Millisecond)
	if p.IsAudioMuted() {
		t.Error("IsAudioMuted = true a while after unmuting a burst")
	}
}

// audiblePage loops a quiet 440 Hz tone in an <audio> element, from a WAV
// it makes, once allowed to: a page may need a click to play (autoplay
// policies), which the button takes.
const audiblePage = `<!doctype html><button id=play>Play</button><p id=state>waiting</p><script>
const rate = 8000, samples = rate;
const buf = new ArrayBuffer(44 + samples * 2), v = new DataView(buf);
const str = (o, s) => [...s].forEach((c, i) => v.setUint8(o + i, c.charCodeAt(0)));
str(0, "RIFF"); v.setUint32(4, 36 + samples * 2, true); str(8, "WAVEfmt ");
v.setUint32(16, 16, true); v.setUint16(20, 1, true); v.setUint16(22, 1, true);
v.setUint32(24, rate, true); v.setUint32(28, rate * 2, true); v.setUint16(32, 2, true);
v.setUint16(34, 16, true); str(36, "data"); v.setUint32(40, samples * 2, true);
for (let i = 0; i < samples; i++) v.setInt16(44 + i * 2, Math.sin(2 * Math.PI * 440 * i / rate) * 3000, true);
window.audio = new Audio(URL.createObjectURL(new Blob([buf], {type: "audio/wav"})));
audio.loop = true;
const play = () => audio.play().then(() => { document.getElementById("state").textContent = "playing"; }, (e) => { document.getElementById("state").textContent = "blocked: " + e.name; });
document.getElementById("play").onclick = play;
play();
</script>`

// A page playing sound is audible, tells it, and stops being audible once
// it stops. It needs an audio device, so it runs only with
// MYGO_E2E_AUDIO=1, by hand: listen for a quiet tone that goes silent for
// the muted part, and click Play if the page is not allowed to play by
// itself.
func TestPageAudible(t *testing.T) {
	if os.Getenv("MYGO_E2E_AUDIO") != "1" {
		t.Skip("plays sound: set MYGO_E2E_AUDIO=1 on a machine with an audio device")
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Audible", Width: 300, Height: 200})
	p := w.Page()
	changes := make(chan bool, 64) // listeners must not block
	defer p.OnAudibleChanged(func(audible bool) { changes <- audible })()
	p.LoadHTML(audiblePage, "")
	t.Log("a quiet tone should play; click Play in the window if it says blocked")
	waitFor(t, w, `document.getElementById("state").textContent === "playing"`)
	if !wait.For(t, "the page playing", changes) || !p.IsAudible() {
		t.Fatalf("the page playing: not audible (IsAudible = %v)", p.IsAudible())
	}

	t.Log("muted: the tone should stop for a while")
	p.SetAudioMuted(true)
	if !p.IsAudioMuted() {
		t.Error("IsAudioMuted = false after SetAudioMuted(true)")
	}
	t.Logf("IsAudible while muted: %v", p.IsAudible())
	listen() // for the tester's ears only: nothing waits on it
	p.SetAudioMuted(false)
	t.Log("unmuted: the tone should play again")
	listen()

	mustEval(t, w, `audio.pause()`)
	t.Log("paused: the tone should stop")
	// Muting may have told changes too, which come first.
	for {
		audible := wait.For(t, "the page stopping", changes)
		if !audible && !p.IsAudible() {
			break
		}
	}
}

// listen gives the tester of TestPageAudible time to hear the page.
func listen() { time.Sleep(2 * time.Second) }
