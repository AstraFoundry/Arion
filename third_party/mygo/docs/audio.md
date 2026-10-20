# Audio

Package `audio` plays sound from Go: effects decoded into memory, music
streamed as it plays, and sound an app makes itself. It mixes everything
playing in Go and sends it to the system's audio output, with no cgo, in
apps of native UI or web pages, on every platform MyGo runs on, browsers
included.

```go
import (
	"github.com/egoist/mygo/audio"
	_ "github.com/egoist/mygo/audio/vorbis" // Ogg Vorbis
)

//go:embed sounds
var sounds embed.FS

data, _ := sounds.ReadFile("sounds/hoe.wav")
hoe, err := audio.DecodeSound(data) // decoded once

hoe.Play() // as often as it sounds, over itself

f, _ := sounds.Open("sounds/spring.ogg")
music, err := audio.OpenStream(f.(io.ReadSeeker)) // decoded as it plays
music.SetLoop(true).SetVolume(0.6).Play()
```

## Sounds and streams

A `Sound` holds decoded samples for effects that play often: `Play` starts
a new player each time, so a sound plays over itself, and `NewPlayer`
returns one paused, to set up before it plays:

```go
hoe.NewPlayer().SetPan(-0.4).SetSpeed(0.95 + rand.Float64()*0.1).Play()
```

`OpenStream` decodes a file about a second ahead as it plays, for music
and other long audio, and `NewPlayer` does the same with any `Decoder`.
Close a stream's player when you are done with it: it closes the file. A
sound's players need no closing.

## Players

A `Player` plays, pauses (`Pause`), and goes back to the start (`Stop`);
`Play` after its end plays it again from the start. `Seek` moves it,
`Position` and `Duration` say where it is and how long it is, and
`Playing` turns false at its end, so a game can wait for a sound.

| setting | effect |
|---|---|
| `SetVolume(v)` | a gain: 1 as decoded, 0 silent |
| `SetPan(p)` | from the left speaker (-1) to the right (1): mono keeps its power, stereo both channels, as Web Audio's `StereoPannerNode` |
| `SetSpeed(s)` | speed and pitch together, as a record's: 2 is twice as fast and an octave higher |
| `SetLoop(true)` | starts again at the end, without a gap |

The setters return the player, to chain them. Volume changes, pauses and
resumes ramp over 5 ms, so none clicks. `audio.SetVolume` sets the volume
of everything.

## Formats

WAV (integer PCM of 8 to 32 bits, 32 or 64 bit floats) decodes out of the
box. Import a format's package for its side effect, as with package
`image`:

| package | format | decoder |
|---|---|---|
| `audio` | WAV | built in |
| `audio/vorbis` | Ogg Vorbis | [jfreymuth/oggvorbis](https://github.com/jfreymuth/oggvorbis) |
| `audio/mp3` | MP3 | [hajimehoshi/go-mp3](https://github.com/hajimehoshi/go-mp3) |

Prefer Ogg Vorbis for music that loops: MP3 files begin and end with the
encoder's padding, a few milliseconds of silence that a loop plays too.

`RegisterFormat` adds a format of your own, and a `Decoder` you write plays
sound an app makes as it plays, such as a synthesizer:

```go
type tone struct{ pos int64 }

func (t *tone) SampleRate() int              { return 48000 }
func (t *tone) Channels() int                { return 1 }
func (t *tone) Len() int64                   { return -1 } // endless
func (t *tone) SetPosition(f int64) error    { t.pos = f; return nil }
func (t *tone) Read(buf []float32) (int, error) {
	for i := range buf {
		buf[i] = 0.2 * float32(math.Sin(2*math.Pi*440*float64(t.pos)/48000))
		t.pos++
	}
	return len(buf), nil
}

audio.NewPlayer(&tone{}).Play()
```

`NewSound` takes samples an app computed once.

## The output

The output opens when something first plays and closes five seconds after
everything stopped, so an app that plays nothing keeps no audio device
busy. It plays at 48 kHz in stereo, about 40 ms ahead; players of other
rates are resampled as they play.

| platform | output |
|---|---|
| macOS | an Audio Queue on the default output, which follows it |
| Linux | PulseAudio's simple API (which PipeWire serves as well), else ALSA's default device |
| Windows | WASAPI in shared mode, opened again when the default output changes |
| browsers | an AudioWorklet; a page may start audio only after the user interacted with it, so it starts on the first click or key |

Where no output opens, as on a machine with no audio device, players still
play on time, unheard, and `audio.Err` says why. Every function and method
is safe to call from any goroutine.

[examples/piano](../examples/piano) is a piano of synthesized keys that
plays from the pointer and the keyboard, with a sustain pedal, dampers that
fade its strings out, and a few tunes it plays by itself.
