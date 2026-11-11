package main

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
)

func TestWorkspaceBuildInputs(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(root, "app")
	writeFiles(t, root, map[string]string{
		"go.work":     "go 1.27\nuse ./app\n",
		"app/go.mod":  "module example.test/app\ngo 1.27\n",
		"app/main.go": "package main\nfunc main() {}\n",
	})
	// Let Go discover the workspace from the app directory, rather than
	// the CLI process's current working directory.
	t.Setenv("GOWORK", "")
	c := &Config{root: app, Main: "."}
	in, err := listBuildInputs(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"go.work", "go.work.sum"} {
		path := filepath.Join(root, name)
		if !slices.Contains(in.files, path) {
			t.Errorf("build inputs omit %s", path)
		}
	}
	before := snapshot(in)
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.27\nuse ./app\n// edited\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.work.sum"), []byte("new checksum\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after := snapshot(in)
	for _, name := range []string{"go.work", "go.work.sum"} {
		path := filepath.Join(root, name)
		if before[path] == after[path] {
			t.Errorf("watcher missed change to %s", name)
		}
	}
	t.Setenv("GOWORK", "off")
	in, err = listBuildInputs(c)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(in.files, filepath.Join(root, "go.work")) {
		t.Error("disabled workspace is watched")
	}
	// Explicit workspace files outside the project's ancestors also count.
	other := filepath.Join(t.TempDir(), "other.work")
	if err := os.WriteFile(other, []byte("go 1.27\nuse "+strconv.Quote(filepath.ToSlash(app))+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", other)
	in, err = listBuildInputs(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{other, other + ".sum"} {
		if !slices.Contains(in.files, path) {
			t.Errorf("explicit workspace input omitted: %s", path)
		}
	}
}
