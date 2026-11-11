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
`Playing` turns false at its end, so a game can wait for a sound;
`OnEnd` calls a function then, on a goroutine of its own, as a playlist
plays the next track (a looping player never ends).

`Position` is where the player is heard. The output mixes ahead of what
the speakers play (see [the output](#the-output)), and `Position` lags
what it mixed by that much, so lyrics, a progress bar or a visualizer
drawn from it keep time with the sound: after `Pause` it goes on while
the output plays what it held, and after `Play` it waits until the
player is heard. `Seek` and `Stop` move it at once.

| setting | effect |
|---|---|
| `SetVolume(v)` | a gain: 1 as decoded, 0 silent |
| `SetPan(p)` | from the left speaker (-1) to the right (1): mono keeps its power, stereo both channels, as Web Audio's `StereoPannerNode` |
| `SetSpeed(s)` | speed and pitch together, as a record's: 2 is twice as fast and an octave higher |
| `SetPreservesPitch(true)` | keeps the pitch at any speed, as HTML media's `preservesPitch`: speech at 1.5× in the voice it was recorded in |
| `SetLoop(true)` | starts again at the end, without a gap |
| `SetLoopRange(start, end)` | loops from `end` back to `start`: a tune's intro plays once, then its loop (`end` 0 is the end) |

The setters return the player, to chain them. Volume changes, pauses and
resumes ramp over 5 ms, so none clicks. `audio.SetVolume` sets the volume
of everything.

`SetSpeed` resamples, so the pitch changes with the speed, which games
rely on for effects that play often. A podcast, lecture or voice-message
player at 1.25×, 1.5× or 2× wants the voice to stay where it is: with
`SetPreservesPitch(true)` the player time-stretches instead (WSOLA, in Go),
playing segments of about 30 ms crossfaded where their waveforms match, so
whole periods are skipped or repeated and the pitch stays:

```go
podcast, err := audio.OpenStream(f)
podcast.SetPreservesPitch(true).SetSpeed(1.5).Play()
```

It suits speech and other pitched sound from about 0.5× to 4×, where sharp
attacks such as drums' come out a little smeared, and it costs about 1% of
a core while a player stretches; players that don't preserve pitch cost
nothing more. Speed changes, seeks, loops and turning it on or off go on
without a gap, and `Position` and `OnEnd` keep the source's time as with
resampling.

## Media keys and Now Playing

`mygo.NowPlaying.Control` lets the media keys and the system's media
controls (Control Center on macOS, the volume flyout on Windows, MPRIS
clients on Linux) play, pause, stop and seek a player, and shows its state
there; see [Now Playing](native.md#now-playing-and-media-keys):

```go
release := mygo.NowPlaying.Control(music)
mygo.NowPlaying.Set(mygo.NowPlayingInfo{Title: "Spring", Artist: "Vivaldi", Commands: mygo.MediaNext})
music.Play()
mygo.NowPlaying.Sync() // after playing it from the app
```

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
encoder's padding, a few milliseconds of silence that a loop plays too,
unless `SetLoopRange` leaves it out.

```go
// 12.5 s of intro, then bars from 12.5 s to 1 min 40 s, again and again.
music.SetLoopRange(12500*time.Millisecond, 100*time.Second).Play()
```

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

`audio.SetLatency` sets how far ahead it mixes, as Web Audio's
`latencyHint`:

| latency | ahead | for |
|---|---|---|
| `LatencyInteractive` (the default) | about 40 ms | games and interfaces: sounds start, stop and change as soon as asked |
| `LatencyPlayback` | about 250 ms | music players: playback rides out the moments the system keeps an app waiting, as macOS does for up to 0.2 s while its appearance changes, where 40 ms runs dry and is heard to stop |

With `LatencyPlayback`, plays, pauses, seeks and volume changes are heard
that much later; players' `Position` follows what is heard. Set it before playing: an output already open opens
again, which drops what it held.

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
