//go:build arion_e2e && windows

package e2e

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode"

	"arion.local/desktop/infrastructure/asr"
	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

// TestWindowsPackagedSpeech exercises the shipped application, native worker,
// real recognition and SendInput delivery into owned Win32 controls. Only the
// microphone and explicit late-result/acknowledgement faults are scripted.
func TestWindowsPackagedSpeech(t *testing.T) {
	packagePath := os.Getenv("ARION_E2E_PACKAGE")
	if packagePath == "" {
		t.Skip("opt-in packaged speech requires ARION_E2E_PACKAGE and ARION_E2E_MODEL_STORE")
	}
	storeRoot := os.Getenv("ARION_E2E_MODEL_STORE")
	if storeRoot == "" {
		t.Fatal("ARION_E2E_MODEL_STORE is required; missing models cannot pass inference acceptance")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(root, "apps/desktop/e2e/testdata/libri-251-137823-0017.wav")
	if hash := speechFileHash(t, fixture); hash != "5f0ef6809a5fcd07764139cfe5b7d67a27a6cd0045dcd48109f1c717d40b66e1" {
		t.Fatal("speech fixture differs from its reviewed development-corpus identity")
	}
	silence := filepath.Join(root, "apps/desktop/e2e/testdata/nonspeech-dev-0.wav")
	if hash := speechFileHash(t, silence); hash != "20eaebffe1816e0ffa6f7f854f5ef4ea80d5349faaf0ce1fec1b713e7fde58fa" {
		t.Fatal("silence fixture differs from its reviewed development-corpus identity")
	}
	// The short path is required by Windows AF_UNIX's sockaddr path limit.
	profile, err := os.MkdirTemp("", "arion-speech-")
	if err != nil {
		t.Fatal(err)
	}
	profile, err = filepath.Abs(profile)
	if err != nil {
		t.Fatal(err)
	}
	tempRoot, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(tempRoot, profile)
	if err != nil || relative == "." || strings.HasPrefix(relative, "..") || filepath.IsAbs(relative) {
		t.Fatal("isolated speech profile must stay inside the temporary directory")
	}
	t.Cleanup(func() { _ = os.RemoveAll(profile) })
	socket := filepath.Join(profile, "control.sock")
	if len(socket) >= 104 {
		t.Fatalf("temporary path exceeds the local socket limit: %s", socket)
	}
	catalog, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	artifact, ok := catalog.Artifact("qwen3-asr-1.7b-gguf-q8")
	if !ok {
		t.Fatal("Windows recognition artifact is absent from the catalog")
	}
	store := provision.NewStore(storeRoot)
	store.Offline = true
	installed, err := store.Installed(artifact)
	if err != nil {
		t.Fatalf("verify source model store: %v", err)
	}
	destination := filepath.Join(profile, "models", artifact.ID)
	if err = os.MkdirAll(destination, 0700); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(installed)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("unexpected directory in verified model store: %s", entry.Name())
		}
		if err = os.Link(filepath.Join(installed, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
			t.Fatalf("link verified model into the isolated profile (same volume required): %v", err)
		}
	}
	output := os.Getenv("ARION_E2E_SPEECH_OUT")
	if output == "" {
		output = filepath.Join(root, ".workspace", "e2e", "windows-speech-"+filepath.Base(profile))
	}
	if err = os.MkdirAll(filepath.Dir(output), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(output, 0700); err != nil {
		t.Fatalf("speech evidence directory must be new: %v", err)
	}
	t.Logf("packaged speech evidence: %s", output)
	trace, err := os.Create(filepath.Join(output, "trace.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = trace.Close() })
	encoder := json.NewEncoder(trace)
	record := func(value any) {
		t.Helper()
		if err := encoder.Encode(value); err != nil {
			t.Fatal(err)
		}
	}
	worker := asr.Locate(filepath.Dir(packagePath))
	record(map[string]any{"event": "provenance", "applicationSha256": speechFileHash(t, packagePath), "selectedWorker": filepath.Base(worker), "workerSha256": speechFileHash(t, worker), "audioSha256": speechFileHash(t, fixture), "modelSha256": artifact.Files["model"].SHA256, "execution": "live", "fixtureOwnership": "isolated", "scope": "packaged recognition and delivery mechanics; one development utterance is not a quality benchmark"})
	stderr, err := os.Create(filepath.Join(output, "app.log"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stderr.Close() })
	process := exec.Command(packagePath)
	process.Env = append(os.Environ(), "ARION_DATA_DIR="+profile, "ARION_E2E_CONTROL="+socket, "ARION_OFFLINE=1", "ARION_ASR_WORKER=")
	process.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	process.Stdout, process.Stderr = stderr, stderr
	if err = process.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- process.Wait() }()
	var conn net.Conn
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopped = true
		if conn != nil {
			_ = conn.SetDeadline(time.Now().Add(time.Second))
			_, _ = io.WriteString(conn, "{\"op\":\"quit\"}\n")
			_ = conn.Close()
		}
		select {
		case exitErr := <-exited:
			if exitErr != nil {
				t.Errorf("owned packaged application exit: %v", exitErr)
			}
			record(map[string]any{"event": "cleanup", "ownedApplicationExited": exitErr == nil})
		case <-time.After(10 * time.Second):
			_ = process.Process.Kill()
			<-exited
			t.Error("owned packaged application did not exit after quit")
		}
	}
	t.Cleanup(stop)
	if !within(15*time.Second, func() bool {
		conn, err = net.DialTimeout("unix", socket, 100*time.Millisecond)
		return err == nil
	}) {
		t.Fatalf("packaged application control socket: %v", err)
	}
	reader := bufio.NewReader(conn)
	call := func(op string, values map[string]string) map[string]any {
		t.Helper()
		request := map[string]string{"op": op}
		for key, value := range values {
			request[key] = value
		}
		if err := conn.SetDeadline(time.Now().Add(2 * time.Minute)); err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(conn).Encode(request); err != nil {
			t.Fatal(err)
		}
		line, err := reader.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		var reply struct {
			Error        string         `json:"error"`
			Observations map[string]any `json:"observations"`
		}
		if err = json.Unmarshal(line, &reply); err != nil {
			t.Fatal(err)
		}
		if op != "state" {
			record(map[string]any{"event": "command", "op": op, "reply": json.RawMessage(line)})
		}
		if reply.Error != "" {
			t.Fatalf("control %s: %s", op, reply.Error)
		}
		return reply.Observations
	}
	state := func(observations map[string]any, key, field string) string {
		object, _ := observations[key].(map[string]any)
		value, _ := object[field].(string)
		return value
	}
	wait := func(key, field, expected string) map[string]any {
		t.Helper()
		var observed map[string]any
		if !within(2*time.Minute, func() bool {
			observed = call("state", nil)
			return state(observed, key, field) == expected
		}) {
			t.Fatalf("wait for %s.%s=%s: %+v", key, field, expected, observed)
		}
		return observed
	}
	wait("model", "state", "ready")
	first, second := startWindowsEditor(t), startWindowsEditor(t)
	reset := func() {
		call("reset", nil)
		first.setText("")
		second.setText("")
		first.focus(t)
	}
	dictate := func(path string) {
		call("start", nil)
		wait("session", "state", "capturing")
		call("play", map[string]string{"path": path})
		call("wait-audio", nil)
		call("stop", nil)
	}
	assertEmpty := func() {
		t.Helper()
		if first.text() != "" || second.text() != "" {
			t.Fatalf("unsafe delivery into owned editors: %q / %q", first.text(), second.text())
		}
	}
	reset()
	dictate(fixture)
	observed := wait("session", "state", "completed")
	if observed["commitCount"] != float64(1) {
		t.Fatalf("commit count: %+v", observed)
	}
	if !within(5*time.Second, func() bool {
		return normalizeSpeechText(first.text()) == "anyhow we want to help got a job for us"
	}) {
		t.Fatalf("recognized text did not reach the owned editor: %q", first.text())
	}
	before := first.characters.Load()
	call("commit-reviewed", nil)
	observed = call("commit-reviewed", nil)
	time.Sleep(200 * time.Millisecond)
	if first.characters.Load() != before || observed["commitCount"] != float64(1) || second.text() != "" {
		t.Fatal("duplicate confirmation delivered text more than once")
	}
	record(map[string]any{"event": "scenario", "id": "offline-recognition-exact-once", "observations": observed, "editorA": first.text(), "editorB": second.text(), "characterCount": first.characters.Load()})
	for _, fault := range []string{"cancel", "focus-change"} {
		reset()
		call("delay-final", nil)
		dictate(fixture)
		wait("session", "state", "finalizing")
		if !within(2*time.Minute, func() bool {
			observed := call("state", nil)
			finalizer, _ := observed["finalizer"].(map[string]any)
			return finalizer["pending"] == true
		}) {
			t.Fatal("real recognition never reached the delayed final-result gate")
		}
		if fault == "cancel" {
			call("cancel", nil)
		} else {
			second.focus(t)
		}
		call("release-final", nil)
		wanted := "cancelled"
		if fault == "focus-change" {
			wanted = "completed"
		}
		observed = wait("session", "state", wanted)
		// A late native result may still be running after cancellation.
		time.Sleep(1500 * time.Millisecond)
		if fault == "cancel" {
			assertEmpty()
			if observed["commitCount"] != float64(0) {
				t.Fatal("cancelled speech attempted insertion")
			}
		}
		if fault == "focus-change" && (state(observed, "commit", "status") == "not_applied" || state(observed, "error", "code") == "TARGET_CHANGED" || first.text() != "" || second.text() == "") {
			t.Fatalf("focus change did not insert into the focused editor: %+v %q %q", observed, first.text(), second.text())
		}
		record(map[string]any{"event": "scenario", "id": fault, "observations": observed, "editorA": first.text(), "editorB": second.text()})
	}
	reset()
	dictate(silence)
	observed = wait("session", "state", "failed")
	assertEmpty()
	if state(observed, "error", "code") != "NO_SPEECH" || observed["commitCount"] != float64(0) {
		t.Fatalf("silence created a delivery candidate: %+v", observed)
	}
	record(map[string]any{"event": "scenario", "id": "silence-no-delivery", "audioSha256": speechFileHash(t, silence), "observations": observed, "editorA": first.text(), "editorB": second.text()})
	reset()
	first.focusControl(t, first.password)
	call("start", nil)
	observed = wait("session", "state", "failed")
	capture, _ := observed["capture"].(map[string]any)
	if state(observed, "error", "code") != "SECURE_FIELD_BLOCKED" || capture["starts"] != float64(0) || observed["commitCount"] != float64(0) || windowText(first.password) != "" {
		t.Fatalf("secure field started capture or committed: %+v", observed)
	}
	record(map[string]any{"event": "scenario", "id": "secure-field-before-capture", "observations": observed})
	stop()
	record(map[string]any{"event": "completed", "scenarios": 5})
	if err = trace.Sync(); err != nil {
		t.Fatal(err)
	}
	identity := speechFileHash(t, trace.Name())
	if err = os.WriteFile(filepath.Join(output, "evidence.json"), []byte(fmt.Sprintf("{\"path\":\"trace.jsonl\",\"sha256\":\"%s\"}\n", identity)), 0600); err != nil {
		t.Fatal(err)
	}
}

func speechFileHash(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func normalizeSpeechText(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}
