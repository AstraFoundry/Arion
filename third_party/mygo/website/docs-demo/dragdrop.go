package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// dragFile is a file dropped on the drag-and-drop demo, as the app read it.
type dragFile struct {
	name, size, line string
}

// dragRead reads the files of a drop: in a browser, from the paths the
// backend keeps them at, as from a disk.
func dragRead(paths []string) []dragFile {
	files := make([]dragFile, 0, len(paths))
	for _, p := range paths {
		f := dragFile{name: filepath.Base(p)}
		b, err := os.ReadFile(p)
		switch {
		case err != nil:
			f.size = err.Error()
		default:
			f.size = dragSize(len(b))
			// The first line of a text: printable, as binary files are not.
			line, _, _ := bytes.Cut(b, []byte("\n"))
			line = bytes.TrimSpace(line)
			if utf8.Valid(line) && !bytes.ContainsFunc(line, func(r rune) bool { return !unicode.IsPrint(r) && r != '\t' }) {
				f.line = string(line)
			}
		}
		files = append(files, f)
	}
	return files
}

func dragSize(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d bytes", n)
	case n < 1000*1000:
		return fmt.Sprintf("%.1f KB", float64(n)/1000)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/1000/1000)
}

func init() {
	var files []dragFile
	demo("drag-and-drop", func(c *ui.Context) {
		t := c.Theme()
		zone := ui.Column(c).Width(440).Height(150).Padding(12, 16).Gap(8).Radius(10).
			Background(t.Surface).Border(1, t.Border).Justify(ui.Center)
		if paths := zone.DroppedFiles(); paths != nil {
			files = dragRead(paths)
		}
		if zone.FileDragOver() {
			zone.Border(2, t.Accent)
		}
		zone.Children(func() {
			if len(files) == 0 {
				ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Icon(c, iconFile).TextColor(t.TextMuted)
					ui.Text(c, "Drop files here").TextColor(t.TextMuted)
				})
				return
			}
			for i, f := range files {
				if i == 3 {
					ui.Text(c, fmt.Sprintf("and %d more", len(files)-i)).FontSize(12).TextColor(t.TextMuted)
					break
				}
				ui.Row(c.Key(i)).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconFile)
					ui.Text(c, f.name).SingleLine().MaxWidth(160)
					ui.Text(c, f.size).FontSize(12).TextColor(t.TextMuted).SingleLine()
					if f.line != "" {
						ui.Text(c, "“"+f.line+"”").FontSize(12).TextColor(t.TextMuted).SingleLine().Grow(1)
					}
				})
			}
		})
		status(c, "Read with os.ReadFile from the paths of the drop")
	})
}
