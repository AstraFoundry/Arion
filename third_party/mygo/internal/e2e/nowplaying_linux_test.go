//go:build linux && (amd64 || arm64)

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// TestNowPlayingMPRIS drives the app's MPRIS player as the desktop's
// media controls do, with gdbus.
func TestNowPlayingMPRIS(t *testing.T) {
	gdbus, err := exec.LookPath("gdbus")
	if err != nil {
		t.Skip("gdbus is needed to act as an MPRIS client")
	}
	call := func(args ...string) (string, error) {
		out, err := exec.Command(gdbus, append([]string{"call", "--session"}, args...)...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	var (
		mu   sync.Mutex
		cmds []string
	)
	off := mygo.NowPlaying.OnCommand(func(cmd mygo.MediaCommand, pos time.Duration) {
		mu.Lock()
		cmds = append(cmds, fmt.Sprint(cmd, " ", pos))
		mu.Unlock()
		if cmd == mygo.MediaPause {
			mygo.NowPlaying.SetPlayback(mygo.PlaybackPaused, 10*time.Second)
		}
	})
	defer off()
	mygo.NowPlaying.Set(mygo.NowPlayingInfo{
		Title: "Spring", Artist: "Vivaldi", Album: "The Four Seasons", Artwork: []byte("\x89PNG\r\n\x1a\n"),
		Duration: 3 * time.Minute, Position: 10 * time.Second, State: mygo.PlaybackPaused,
		Commands: mygo.MediaPlay | mygo.MediaPause | mygo.MediaNext | mygo.MediaSeek,
	})
	defer mygo.NowPlaying.Clear()

	var name string
	eventually(t, "the player's bus name", func() bool {
		out, _ := call("-d", "org.freedesktop.DBus", "-o", "/org/freedesktop/DBus", "-m", "org.freedesktop.DBus.ListNames")
		// A list of quoted names, as (['org.freedesktop.DBus', ':1.0'],)
		name = regexp.MustCompile(fmt.Sprintf(`org\.mpris\.MediaPlayer2\.[^']+\.instance%d\b`, os.Getpid())).FindString(out)
		return name != ""
	})
	player := func(args ...string) string {
		t.Helper()
		out, err := call(append([]string{"-d", name, "-o", "/org/mpris/MediaPlayer2", "-m"}, args...)...)
		if err != nil {
			t.Fatalf("%v: %v: %s", args, err, out)
		}
		return out
	}
	get := func(prop string) string {
		t.Helper()
		return player("org.freedesktop.DBus.Properties.Get", "org.mpris.MediaPlayer2.Player", prop)
	}

	if got := get("PlaybackStatus"); got != "(<'Paused'>,)" {
		t.Errorf("PlaybackStatus = %s", got)
	}
	if got := get("Position"); got != "(<int64 10000000>,)" {
		t.Errorf("Position = %s, want 10 s while paused", got)
	}
	meta := get("Metadata")
	for _, want := range []string{"'xesam:title': <'Spring'>", "'xesam:artist': <['Vivaldi']>", "'xesam:album': <'The Four Seasons'>", "'mpris:length': <int64 180000000>", "'mpris:artUrl': <'file://"} {
		if !strings.Contains(meta, want) {
			t.Errorf("Metadata %s lacks %s", meta, want)
		}
	}
	for prop, want := range map[string]string{"CanPlay": "true", "CanPause": "true", "CanGoNext": "true", "CanGoPrevious": "false", "CanSeek": "true"} {
		if got := get(prop); got != "(<"+want+">,)" {
			t.Errorf("%s = %s, want %s", prop, got, want)
		}
	}

	player("org.mpris.MediaPlayer2.Player.PlayPause")              // paused: play
	player("org.mpris.MediaPlayer2.Player.Previous")               // not offered
	player("org.mpris.MediaPlayer2.Player.Seek", "5000000")        // 10 s + 5 s
	player("org.mpris.MediaPlayer2.Player.SetPosition", "/x", "1") // another track
	track := meta[strings.Index(meta, "/org/mygo/track/"):]
	track = track[:strings.IndexByte(track, '\'')]
	player("org.mpris.MediaPlayer2.Player.SetPosition", track, "60000000")
	want := []string{
		fmt.Sprint(mygo.MediaPlay, " 10s"), // where it was paused
		fmt.Sprint(mygo.MediaSeek, " 15s"),
		fmt.Sprint(mygo.MediaSeek, " 1m0s"),
	}
	eventually(t, "the commands", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(cmds) >= len(want)
	})
	mu.Lock()
	got := strings.Join(cmds, ", ")
	mu.Unlock()
	if got != strings.Join(want, ", ") {
		t.Errorf("commands = %s, want %s", got, strings.Join(want, ", "))
	}

	mygo.NowPlaying.Clear()
	eventually(t, "the player leaving the bus", func() bool {
		out, _ := call("-d", "org.freedesktop.DBus", "-o", "/org/freedesktop/DBus", "-m", "org.freedesktop.DBus.NameHasOwner", name)
		return out == "(false,)"
	})
}
