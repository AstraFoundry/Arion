package mygo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateExecutableHelper(t *testing.T) {
	if os.Getenv("MYGO_UPDATE_EXECUTABLE_HELPER") != "1" {
		return
	}
	var err error
	startExe, err = os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(startExe)
	if err != nil {
		t.Fatal(err)
	}
	cleanUpdateLeftovers()
	fmt.Println(string(contents[len(contents)-3:]))
	_, _ = os.Stdin.Read(make([]byte, 1))
}

func TestSwapFilesReplacesRunningExecutableAndCleansAtRelaunch(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	target, staged := t.TempDir(), t.TempDir()
	const name = "Update Test.exe"
	for directory, version := range map[string]string{target: "old", staged: "new"} {
		if err := os.WriteFile(filepath.Join(directory, name), append(contents[:len(contents):len(contents)], version...), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	start := func(want string) func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		cmd := exec.CommandContext(ctx, filepath.Join(target, name), "-test.run=^TestUpdateExecutableHelper$")
		cmd.Env = append(os.Environ(), "MYGO_UPDATE_EXECUTABLE_HELPER=1")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		if err := cmd.Start(); err != nil {
			cancel()
			t.Fatal(err)
		}
		t.Cleanup(cancel)
		line, err := bufio.NewReader(stdout).ReadString('\n')
		if err != nil || line != want+"\n" {
			cancel()
			t.Fatalf("launched version = %q, %v; want %q", line, err, want)
		}
		return func() {
			_ = stdin.Close()
			_, _ = io.Copy(io.Discard, stdout)
			if err := cmd.Wait(); err != nil {
				t.Fatal(err)
			}
			cancel()
		}
	}
	stopOld := start("old")
	if err := swapFiles(staged, target); err != nil {
		t.Fatal(err)
	}
	leftovers, err := filepath.Glob(filepath.Join(target, ".*.old-*"))
	if err != nil || len(leftovers) != 1 {
		t.Fatalf("running old executable = %v, %v", leftovers, err)
	}
	stopOld()
	stopNew := start("new")
	stopNew()
	if leftovers, err := filepath.Glob(filepath.Join(target, ".*.old-*")); err != nil || len(leftovers) != 0 {
		t.Fatalf("relaunch left old executable = %v, %v", leftovers, err)
	}
}

func TestSwapFilesRollsBackAddedEntriesWhenReplacementIsLocked(t *testing.T) {
	target, staged := t.TempDir(), t.TempDir()
	write := func(root, name, contents string) {
		t.Helper()
		file := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(target, "b-existing", "old")
	write(target, "z-locked", "locked old")
	write(target, "Uninstall.exe", "installer-owned")
	write(staged, "a-added/resource", "new resource")
	write(staged, "b-existing", "new")
	write(staged, "c-added", "new file")
	write(staged, "z-locked", "locked new")
	locked, err := os.Open(filepath.Join(target, "z-locked"))
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Close()
	if err := swapFiles(staged, target); err == nil {
		t.Fatal("replacement of a locked file succeeded")
	}
	for _, name := range []string{"a-added", "c-added"} {
		if _, err := os.Stat(filepath.Join(target, name)); !os.IsNotExist(err) {
			t.Errorf("failed update retained %s: %v", name, err)
		}
	}
	for name, want := range map[string]string{"b-existing": "old", "z-locked": "locked old", "Uninstall.exe": "installer-owned"} {
		got, err := os.ReadFile(filepath.Join(target, name))
		if err != nil || string(got) != want {
			t.Errorf("rollback %s = %q, %v; want %q", name, got, err, want)
		}
	}
}
