//go:build !js

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// autoplayPage plays 30 seconds of silence, unmuted, as soon as it loads,
// with no user gesture, and tells in state whether play() resolved. The
// track is silent so that the test makes no sound, but it is an audio
// track all the same, which the autoplay policies decide on; they never
// look at the samples. pauses counts the pause events, as a policy that
// lets play() resolve may pause the element once it finds its sound.
const autoplayPage = `<!doctype html><p>Autoplay</p><script>
const rate = 8000, samples = rate * 30;
const buf = new ArrayBuffer(44 + samples * 2), v = new DataView(buf);
const str = (o, s) => [...s].forEach((c, i) => v.setUint8(o + i, c.charCodeAt(0)));
str(0, "RIFF"); v.setUint32(4, 36 + samples * 2, true); str(8, "WAVEfmt ");
v.setUint32(16, 16, true); v.setUint16(20, 1, true); v.setUint16(22, 1, true);
v.setUint32(24, rate, true); v.setUint32(28, rate * 2, true); v.setUint16(32, 2, true);
v.setUint16(34, 16, true); str(36, "data"); v.setUint32(40, samples * 2, true);
window.state = "waiting"; window.pauses = 0;
window.audio = new Audio(URL.createObjectURL(new Blob([buf], {type: "audio/wav"})));
audio.onpause = () => pauses++;
audio.play().then(() => { state = "playing"; }, (e) => { state = "blocked: " + e.name; });
</script>`

// autoplayState is what a page of autoplayPage did.
type autoplayState struct {
	State       string  `json:"state"`
	CurrentTime float64 `json:"currentTime"`
	Paused      bool    `json:"paused"`
	Muted       bool    `json:"muted"`
	Pauses      int     `json:"pauses"`
}

// played reports whether the page plays its unmuted audio.
func (s autoplayState) played() bool {
	return s.State == "playing" && !s.Paused && !s.Muted && s.CurrentTime >= autoplayPlayed
}

// autoplayPlayed is the time, in seconds of the track, that a playing
// page reaches, which its clock only reaches if it plays.
const autoplayPlayed = 0.5

// autoplayIn shows autoplayPage in a hidden window, which gets no user
// gesture, with autoplay, and returns what it did once play() settled and,
// had it resolved, once the audio played on or paused.
func autoplayIn(t *testing.T, autoplay mygo.Autoplay) autoplayState {
	t.Helper()
	w := newWindow(t, mygo.WindowOptions{Title: "Autoplay", Hidden: true, Width: 300, Height: 200,
		Page: mygo.PageOptions{Autoplay: autoplay}})
	w.Page().LoadHTML(autoplayPage, "")
	waitFor(t, w, `window.state && state !== "waiting"`)
	waitFor(t, w, `state !== "playing" || audio.paused || audio.currentTime >= 0.5`)
	v, err := mygo.EvalAs[string](w.Page(), `JSON.stringify({state, currentTime: audio.currentTime, paused: audio.paused, muted: audio.muted, pauses})`)
	if err != nil {
		t.Fatal(err)
	}
	var s autoplayState
	if err := json.Unmarshal([]byte(v), &s); err != nil {
		t.Fatalf("%s: %v", v, err)
	}
	return s
}

// TestAutoplay plays audio, unmuted, from the page of a hidden window,
// which no user gesture reaches: with AutoplayAllow, play() resolves and
// the audio's clock goes on. Without it, WebKitGTK refuses the page
// (NotAllowedError); what WKWebView and WebView2 do then is only logged,
// as their policies have not been checked to refuse this page on the CI
// runners.
//
// On Windows the policy is the argument of WebView2's browser, which the
// first page decides: AutoplayAllow runs in a test process whose first
// pages ask for it, with a user data folder of its own, as a browser
// running with other arguments refuses it this process's.
func TestAutoplay(t *testing.T) {
	child := os.Getenv("MYGO_E2E_AUTOPLAY") != ""
	if !child {
		s := autoplayIn(t, mygo.AutoplayUserGesture)
		t.Logf("AutoplayUserGesture: %+v", s)
		if runtime.GOOS == "linux" && (s.played() || !strings.HasPrefix(s.State, "blocked") && !s.Paused) {
			t.Errorf("AutoplayUserGesture: the page played unmuted audio without a gesture: %+v", s)
		}
	}
	if runtime.GOOS == "windows" && !child {
		ctx, cancel := context.WithTimeout(t.Context(), 4*wait.Guard)
		defer cancel()
		data, err := os.MkdirTemp("", "mygo-autoplay")
		if err != nil {
			t.Fatal(err)
		}
		// The browser may hold the folder a moment after the test ends.
		defer os.RemoveAll(data)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAutoplay$", "-test.v")
		cmd.Env = append(os.Environ(), "MYGO_E2E_AUTOPLAY=1", "LOCALAPPDATA="+data)
		out, err := cmd.CombinedOutput()
		t.Logf("%s", out)
		// The child's windows had the foreground, which the tests after
		// expect this process's to get.
		takeForeground(newWindow(t, mygo.WindowOptions{Title: "front", Width: 120, Height: 80}))
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	s := autoplayIn(t, mygo.AutoplayAllow)
	t.Logf("AutoplayAllow: %+v", s)
	if !s.played() {
		t.Errorf("AutoplayAllow: the page did not play unmuted audio without a gesture: %+v", s)
	}
}
