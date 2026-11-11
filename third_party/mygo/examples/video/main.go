// Video is an animated background in native UI: a short GIF that loops
// behind the window's content, as a launcher's library or a music
// player's hero shows one.
//
//	go run ./examples/video
//
// The clip's frames decode on a goroutine a few ahead, and each reaches
// the screen as it comes due, so the window draws one frame for each of
// the clip's and none between. The clip stands still while the window is
// out of sight, and shows its first frame while the desktop asks for less
// motion.
package main

import (
	_ "embed"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed waves.gif
var wavesGIF []byte

// app is the state the window shows.
type app struct {
	waves *ui.VideoPlayer
}

func newApp() (*app, error) {
	d, err := ui.NewGIFDecoder(wavesGIF)
	if err != nil {
		return nil, err
	}
	waves, err := ui.NewVideoPlayer(d)
	if err != nil {
		return nil, err
	}
	return &app{waves: waves.SetLoop(true)}, nil
}

// view builds the interface from the state, on the main thread, whenever
// the window needs a frame.
func (s *app) view(c *ui.Context) {
	ui.Box(c).Fill().Children(func() {
		ui.Video(c, s.waves).Fill().Fit(ui.Cover)
		// Over the clip, which takes the window.
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Gap(16).Children(func() {
			ui.Text(c, "Wavelength").FontSize(44).Bold().TextColor(ui.RGB(255, 255, 255))
			label := "Pause"
			if !s.waves.Playing() {
				label = "Play"
			}
			if ui.Button(c.Key("play"), label).Width(96).Clicked() {
				if s.waves.Playing() {
					s.waves.Pause()
				} else {
					s.waves.Play()
				}
			}
		})
	})
}

func main() {
	s, err := newApp()
	if err != nil {
		log.Fatal(err)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Video",
			Width:   640,
			Height:  360,
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
