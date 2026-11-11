# Video

<!-- preview video 240 -->

`ui.Video` shows a video that plays: a short clip looping behind the
content, as a launcher's library or a music player's hero shows one, or a
preview in a grid. It shows a `ui.VideoPlayer`, which plays the frames a
decoder gives it, and is laid out, fitted and rounded as an
[Image](image.md) is.

```go
//go:embed waves.gif
var wavesGIF []byte

d, err := ui.NewGIFDecoder(wavesGIF)
if err != nil {
	log.Fatal(err)
}
app.waves, err = ui.NewVideoPlayer(d)
if err != nil {
	log.Fatal(err)
}
app.waves.SetLoop(true)

// In the view:
ui.Video(c, app.waves).Fill().Fit(ui.Cover)
```

The [video example](../../examples/video) plays a clip behind a window's
content.

## Players

`ui.NewVideoPlayer` decodes the first frame at once, which `Video` shows
from its first frame and which reports a video that does not decode, so
make players once, not in the view, as bitmaps. A new player plays as soon
as a `Video` shows it. `Pause` holds it at the frame it shows, and `Play`
goes on from there, or plays it again from its start once it ended;
`Playing` tells which. `SetLoop` plays it again each time it ends, without
a gap. A video that ends shows its last frame, and `Err` tells what
stopped one before its end. `Close` stops decoding and closes the decoder,
when it is an `io.Closer`. Call a player's methods on the main thread, in
the view or an input handler, and show it in one `Video` at a time.

```go
label := "Pause"
if !app.waves.Playing() {
	label = "Play"
}
if ui.Button(c.Key("play"), label).Clicked() {
	if app.waves.Playing() {
		app.waves.Pause()
	} else {
		app.waves.Play()
	}
}
```

Videos have no sound.

## Playing

A player decodes its frames on a goroutine of its own, a couple ahead of
the one showing, into frames it uses again, and each frame reaches the
screen through a [bitmap's `Update`](image.md#bitmaps) as it comes due:
the window draws one frame for each of the video's, which `After` asks
for, and none between. When the decoder falls behind, the window waits
for it without drawing, and of the frames that came due meanwhile shows
only the last, so the video keeps to the clock rather than lag behind it.

A video moves on only while the view builds it and some of the window
shows. One that went out of the view, as on another tab, or in a window
that is minimized, hidden or covered by other windows (on macOS, and in
browsers), goes on from where it stood once it shows again, and decodes
nothing meanwhile. While the desktop asks for less motion
(`Preferences.ReduceMotion`), a video shows its first frame and does not
play; it goes on from where it stood once the desktop no longer asks.

## Decoders

`ui.NewGIFDecoder` decodes animated GIFs, in Go, as browsers show them:
each frame drawn over what the ones before it left, as its disposal says,
and a frame of 10 ms or less shown for 100 ms. The GIF decodes into memory
at once, a byte a pixel for each frame, and each frame is drawn as it
plays. It plays once or loops as `SetLoop` says, whatever loop count the
GIF gives.

A `ui.VideoDecoder` of your own plays another format: `Next` returns the
next frame, a picture of any image type and how long it shows, or `io.EOF`
after the last, and `Rewind` goes back to before the first. The player
copies each picture, so a decoder may draw every frame into the same one,
and calls the decoder from one goroutine at a time.

```go
type frames struct {
	pics []*image.RGBA
	next int
}

func (f *frames) Next() (ui.VideoFrame, error) {
	if f.next == len(f.pics) {
		return ui.VideoFrame{}, io.EOF
	}
	f.next++
	return ui.VideoFrame{Image: f.pics[f.next-1], Duration: time.Second / 30}, nil
}

func (f *frames) Rewind() error { f.next = 0; return nil }
```

Decoders of the system's codecs (H.264, HEVC, VP9 and AV1 through Media
Foundation, AVFoundation and GStreamer) are not part of MyGo yet.

## Accessibility

Videos are decorations that assistive technology does not see, unless
`Label` names them, as images are.
