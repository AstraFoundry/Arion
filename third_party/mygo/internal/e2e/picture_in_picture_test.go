//go:build !js

package e2e

import (
	"os"
	"runtime"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
)

// pipPage plays a video with sound, which it records from a canvas and a
// quiet tone first, at app://localhost/pip: large and with sound, as the
// video WebKit picks for its media controls is. window.restored tells
// that the page came back from the back/forward cache. A page may need a
// click to play (autoplay policies), which the button takes.
const pipPage = `<!doctype html><button id=play>Play</button><p id=state>recording</p>
<video id=video width=640 height=360 loop playsinline></video><script>
addEventListener("pageshow", (e) => { window.restored = e.persisted; });
const canvas = Object.assign(document.createElement("canvas"), {width: 320, height: 180}), g = canvas.getContext("2d");
let frame = 0;
const draw = () => { g.fillStyle = "hsl(" + (frame++ * 6 % 360) + " 70% 50%)"; g.fillRect(0, 0, 320, 180); requestAnimationFrame(draw); };
draw();
const audio = new AudioContext(), tone = audio.createOscillator(), gain = audio.createGain(), out = audio.createMediaStreamDestination();
gain.gain.value = 0.02;
tone.connect(gain).connect(out);
tone.start();
const recorder = new MediaRecorder(new MediaStream([...canvas.captureStream(30).getVideoTracks(), ...out.stream.getAudioTracks()])), chunks = [];
const play = () => video.play().then(() => { state.textContent = "playing"; }, (e) => { state.textContent = "blocked: " + e.name; });
recorder.ondataavailable = (e) => chunks.push(e.data);
recorder.onstop = () => { video.src = URL.createObjectURL(new Blob(chunks, {type: recorder.mimeType})); play(); };
recorder.start();
setTimeout(() => recorder.stop(), 2000);
document.getElementById("play").onclick = () => { audio.resume(); play(); };
</script>`

// A page without a video has none in picture in picture, and toggling
// moves nothing.
func TestPictureInPictureWithoutVideo(t *testing.T) {
	w := newWindow(t, mygo.WindowOptions{Title: "No video", Width: 300, Height: 200, Page: mygo.PageOptions{PictureInPicture: true}})
	p := w.Page()
	p.LoadHTML(`<p>no video</p>`, "")
	waitFor(t, w, `document.body.textContent === "no video" && document.readyState === "complete"`)
	if p.IsPictureInPictureActive() {
		t.Error("IsPictureInPictureActive = true for a page without a video")
	}
	p.TogglePictureInPicture()
	mustEval(t, w, `1`) // a round trip, for the toggle to land
	if p.IsPictureInPictureActive() {
		t.Error("IsPictureInPictureActive = true after toggling a page without a video")
	}
}

// TogglePictureInPicture moves a playing video into picture in picture
// and back, and a page that leaves with its video out comes back without
// it. It opens the system's picture in picture window, which needs a
// display, and its page plays sound, so it runs only with
// MYGO_E2E_PIP=1, by hand: click Play if the page is not allowed to play
// by itself, and watch the video float out and come back.
func TestPictureInPicture(t *testing.T) {
	if os.Getenv("MYGO_E2E_PIP") != "1" {
		t.Skip("opens a picture in picture window: set MYGO_E2E_PIP=1 on a machine with a display")
	}
	if runtime.GOOS == "linux" {
		t.Skip("WebKitGTK has no picture in picture")
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Picture in picture", Width: 700, Height: 480, Page: mygo.PageOptions{PictureInPicture: true}})
	p := w.Page()
	if err := p.LoadURL("app://localhost/pip"); err != nil {
		t.Fatal(err)
	}
	t.Log("a video should play; click Play in the window if it says blocked")
	waitFor(t, w, `document.getElementById("state").textContent === "playing"`)
	active := func(want bool) func() bool {
		return func() bool { return p.IsPictureInPictureActive() == want }
	}

	p.TogglePictureInPicture()
	wait.Until(t, "the video in picture in picture", active(true))
	p.TogglePictureInPicture()
	wait.Until(t, "the video back in the page", active(false))

	// WebKit restores a page from its back/forward cache believing that
	// its video is still out, unless it was kept out of the cache.
	p.TogglePictureInPicture()
	wait.Until(t, "the video in picture in picture again", active(true))
	mustEval(t, w, `location.href = "/"`)
	waitFor(t, w, `location.pathname === "/" && document.readyState === "complete"`)
	wait.Until(t, "no video in picture in picture on the next page", active(false))
	p.GoBack()
	waitFor(t, w, `location.pathname === "/pip" && document.readyState === "complete"`)
	restored := mustEval(t, w, `window.restored === true`)
	if p.IsPictureInPictureActive() || mustEval(t, w, `document.pictureInPictureElement !== null`) == true {
		t.Errorf("the page came back believing its video is in picture in picture (restored from the cache: %v)", restored)
	}
	if runtime.GOOS == "darwin" && restored == true {
		t.Error("WebKit kept a page that left with its video in picture in picture in its back/forward cache")
	}
}
