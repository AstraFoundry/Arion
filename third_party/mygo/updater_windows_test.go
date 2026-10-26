package mygo

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/update"
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

func TestPreparedUpdateReplacesRunningExecutableAndCleansAtRelaunch(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	// Match installTarget in the relaunched process, which resolves 8.3 paths.
	target, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	staged := t.TempDir()
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
	prepared := prepareWindowsUpdate(t, staged, target)
	if err := prepared.Apply(); err != nil {
		t.Fatal(err)
	}
	if err := prepared.Acknowledge(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(prepared.work, "backup", name)); err != nil {
		t.Fatalf("running old executable not preserved: %v", err)
	}
	stopOld()
	stopNew := start("new")
	stopNew()
	if _, err := os.Stat(prepared.work); !os.IsNotExist(err) {
		t.Fatalf("relaunch left acknowledged update stage: %v", err)
	}
}

func TestPreparedUpdateRollsBackAddedEntriesWhenReplacementIsLocked(t *testing.T) {
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
	prepared := prepareWindowsUpdate(t, staged, target)
	locked, err := os.Open(filepath.Join(target, "z-locked"))
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Close()
	if err := prepared.Apply(); err == nil {
		t.Fatal("replacement of a locked file succeeded")
	}
	if err := prepared.Discard(); err != nil {
		t.Fatal(err)
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

// prepareWindowsUpdate stages a signed archive for the native Windows lock regressions.
func prepareWindowsUpdate(t *testing.T, source, target string) *PreparedUpdate {
	t.Helper()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	var archive bytes.Buffer
	if err := update.WriteArchive(&archive, source, names); err != nil {
		t.Fatal(err)
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive.Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive.Bytes()) }))
	t.Cleanup(server.Close)
	manifest := update.Manifest{Version: "2.0.0", URL: server.URL + "/app.tar.gz", Size: int64(archive.Len()), Signature: update.Sign(priv, sum[:])}
	prepared, err := prepareUpdate(context.Background(), manifest, base64.StdEncoding.EncodeToString(pub), "1.0.0", target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return prepared
}
