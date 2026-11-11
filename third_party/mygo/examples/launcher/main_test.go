package main

import (
	"slices"
	"testing"

	"github.com/egoist/mygo"
)

func TestFilter(t *testing.T) {
	l := &launcher{}
	for _, name := range []string{"Calendar", "Visual Studio Code", "Activity Monitor", "Code Editor", "Xcode", "Calculator"} {
		l.apps = append(l.apps, mygo.InstalledApp{Name: name, Path: "/" + name})
	}
	for query, want := range map[string][]string{
		"":     {"Calendar", "Visual Studio Code", "Activity Monitor", "Code Editor", "Xcode", "Calculator"},
		"cal":  {"Calendar", "Calculator"},
		"code": {"Code Editor", "Visual Studio Code", "Xcode"},
		"vsc":  {"Visual Studio Code"},
		"ACT":  {"Activity Monitor", "Calculator"}, // its letters in order, last
		"zzz":  nil,
	} {
		l.query = query
		l.filter()
		var got []string
		for _, a := range l.results {
			got = append(got, a.Name)
		}
		if !slices.Equal(got, want) {
			t.Errorf("%q: %v, want %v", query, got, want)
		}
	}
}
