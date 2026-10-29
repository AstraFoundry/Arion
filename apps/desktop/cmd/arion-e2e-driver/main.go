//go:build darwin

// Command arion-e2e-driver runs owned product scenarios against the packaged
// Arion.app and an isolated AppKit editor fixture on the actual OS. It
// speaks the tools/driver JSONL protocol; assertions stay in the harness.
package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"arion.local/desktop/infrastructure/provision"
)

type driverConfig struct {
	Artifacts []struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	} `json:"artifacts"`
	App      string            `json:"app"`
	Fixture  string            `json:"fixtureApp"`
	Models   string            `json:"models"`
	Audio    map[string]string `json:"audio"`
	Evidence string            `json:"evidence"`
}

type request struct {
	Protocol  int              `json:"protocol"`
	RequestID string           `json:"requestId"`
	Op        string           `json:"op"`
	ID        string           `json:"id"`
	Actions   []map[string]any `json:"actions"`
}

var capabilities = []string{"shortcut", "audio-fixture", "target-editor", "secure-editor", "network-monitor",
	"delayed-inference", "commit-ack-fault", "input-mode"}

type client struct {
	conn    net.Conn
	scanner *bufio.Scanner
}

func dial(path string, timeout time.Duration) (*client, error) {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.Dial("unix", path)
		if err == nil {
			scanner := bufio.NewScanner(conn)
			scanner.Buffer(make([]byte, 1<<20), 8<<20)
			return &client{conn: conn, scanner: scanner}, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *client) call(command map[string]any) (map[string]any, error) {
	data, _ := json.Marshal(command)
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	if !c.scanner.Scan() {
		return nil, errors.New("connection closed")
	}
	var reply map[string]any
	if err := json.Unmarshal(c.scanner.Bytes(), &reply); err != nil {
		return nil, err
	}
	if message, ok := reply["error"].(string); ok && message != "" {
		return reply, fmt.Errorf("%v: %s", command["op"], message)
	}
	return reply, nil
}

type driver struct {
	dir       string
	config    driverConfig
	profile   string
	arion     *client
	editor    *client
	arionPID  int
	trace     []map[string]any
	traceMu   sync.Mutex
	netSample map[string]bool
}

func (d *driver) record(kind string, value any) {
	d.traceMu.Lock()
	d.trace = append(d.trace, map[string]any{"t": time.Now().UnixMilli(), "kind": kind, "value": value})
	d.traceMu.Unlock()
}

func (d *driver) path(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(d.dir, p)
}

// hello attests artifacts, creates an isolated profile and launches the
// actual app and editor fixture as their own processes.
func (d *driver) hello() (map[string]any, error) {
	attested := map[string]string{}
	for _, artifact := range d.config.Artifacts {
		digest, err := provision.HashFile(d.path(artifact.Path))
		if err != nil {
			return nil, err
		}
		attested[artifact.ID] = digest
	}
	profile, err := os.MkdirTemp("", "arion-e2e-profile-")
	if err != nil {
		return nil, err
	}
	d.profile = profile
	// The isolated profile reuses verified model files read-only and starts
	// with onboarding complete; it never touches the user's own profile.
	if err := os.Symlink(d.path(d.config.Models), filepath.Join(profile, "models")); err != nil {
		return nil, err
	}
	prefs := map[string]any{"version": 1, "onboardingCompleted": true, "mode": "shortcut",
		"delivery": "automatic", "sounds": false}
	data, _ := json.Marshal(prefs)
	if err := os.WriteFile(filepath.Join(profile, "preferences.json"), data, 0o600); err != nil {
		return nil, err
	}
	arionSocket := filepath.Join(profile, "control.sock")
	editorSocket := filepath.Join(profile, "editor.sock")
	if err := exec.Command("/usr/bin/open", "-n", "-g", d.path(d.config.Fixture), "--env", "ARION_FIXTURE_SOCKET="+editorSocket).Run(); err != nil {
		return nil, fmt.Errorf("launch editor fixture: %w", err)
	}
	if err := exec.Command("/usr/bin/open", "-n", "-g", d.path(d.config.App), "--env", "ARION_DATA_DIR="+profile,
		"--env", "ARION_E2E_CONTROL="+arionSocket, "--env", "ARION_OFFLINE=1").Run(); err != nil {
		return nil, fmt.Errorf("launch Arion: %w", err)
	}
	if d.editor, err = dial(editorSocket, 20*time.Second); err != nil {
		return nil, fmt.Errorf("editor fixture socket: %w", err)
	}
	if d.arion, err = dial(arionSocket, 30*time.Second); err != nil {
		return nil, fmt.Errorf("Arion control socket: %w", err)
	}
	reply, err := d.arion.call(map[string]any{"op": "hello"})
	if err != nil {
		return nil, err
	}
	d.arionPID = int(reply["pid"].(float64))
	// The model loads from the local verified store; wait until it is ready.
	deadline := time.Now().Add(90 * time.Second)
	for {
		state, err := d.arion.call(map[string]any{"op": "state"})
		if err != nil {
			return nil, err
		}
		model := state["observations"].(map[string]any)["model"].(map[string]any)
		if model["state"] == "ready" {
			d.record("model", model)
			break
		}
		if model["state"] == "failed" || time.Now().After(deadline) {
			return nil, fmt.Errorf("model not ready: %v", model)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return map[string]any{"execution": "live", "artifacts": attested, "platform": "darwin", "capabilities": capabilities,
		"hardware": map[string]any{"os": "darwin"}, "process": map[string]any{"arionPid": d.arionPID}}, nil
}

// sampleNetwork records the app's internet sockets; lsof -i lists TCP/UDP
// endpoints only, so the local control socket is excluded.
func (d *driver) sampleNetwork() {
	out, _ := exec.Command("/usr/sbin/lsof", "-a", "-i", "-n", "-P", "-p", fmt.Sprint(d.arionPID)).Output()
	for _, line := range strings.Split(string(out), "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 9 {
			d.netSample[fields[7]+" "+fields[8]] = true
		}
	}
}

func (d *driver) editorState() (map[string]any, error) {
	return d.editor.call(map[string]any{"op": "state"})
}

func (d *driver) arionState() (map[string]any, error) {
	reply, err := d.arion.call(map[string]any{"op": "state"})
	if err != nil {
		return nil, err
	}
	return reply["observations"].(map[string]any), nil
}

func (d *driver) waitSession(want ...string) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		state, err := d.arionState()
		if err != nil {
			return err
		}
		current := state["session"].(map[string]any)["state"]
		for _, w := range want {
			if current == w {
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("session never reached %v", want)
}

var terminal = []string{"completed", "failed", "cancelled"}

func (d *driver) action(action map[string]any) error {
	op, _ := action["op"].(string)
	d.record("action", action)
	call := func(command map[string]any) error {
		reply, err := d.arion.call(command)
		d.record("arion", reply["observations"])
		return err
	}
	switch op {
	case "reset-owned-fixtures":
		if _, err := d.editor.call(map[string]any{"op": "reset"}); err != nil {
			return err
		}
		if err := call(map[string]any{"op": "reset"}); err != nil {
			return err
		}
		if err := call(map[string]any{"op": "set-mode", "mode": "shortcut"}); err != nil {
			return err
		}
		return call(map[string]any{"op": "reset"})
	case "disable-network":
		// The app runs with ARION_OFFLINE=1 (no downloads); sockets are
		// observed independently with lsof.
		return nil
	case "focus-editor", "focus-secure-editor":
		target, _ := action["id"].(string)
		if op == "focus-secure-editor" {
			target = "secure"
		}
		// LaunchServices activation is honored where an app's own
		// activation request may be refused by cooperative activation.
		if err := exec.Command("/usr/bin/open", d.path(d.config.Fixture)).Run(); err != nil {
			return fmt.Errorf("activate editor fixture: %w", err)
		}
		if _, err := d.editor.call(map[string]any{"op": "focus", "target": target}); err != nil {
			return err
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			state, err := d.editorState()
			if err != nil {
				return err
			}
			if state["focused"] == target && state["active"] == true {
				return nil
			}
			time.Sleep(50 * time.Millisecond)
		}
		return fmt.Errorf("editor %s never focused", target)
	case "toggle-recording":
		if err := call(map[string]any{"op": "toggle"}); err != nil {
			return err
		}
		return d.waitSession(append([]string{"capturing"}, terminal...)...)
	case "play-audio":
		fixture := d.config.Audio[fmt.Sprint(action["fixture"])]
		if fixture == "" {
			return fmt.Errorf("unknown audio fixture %v", action["fixture"])
		}
		if err := call(map[string]any{"op": "play", "path": d.path(fixture)}); err != nil {
			return err
		}
		return call(map[string]any{"op": "wait-audio"})
	case "stop-recording":
		return call(map[string]any{"op": "stop"})
	case "dictate-fixture":
		for _, step := range []map[string]any{{"op": "toggle-recording"}, {"op": "play-audio", "fixture": action["fixture"]}, {"op": "stop-recording"}} {
			if err := d.action(step); err != nil {
				return err
			}
		}
		return nil
	case "delay-final-callback":
		return call(map[string]any{"op": "delay-final"})
	case "release-final-callback":
		return call(map[string]any{"op": "release-final"})
	case "lose-commit-ack":
		return call(map[string]any{"op": "lose-ack"})
	case "cancel":
		return call(map[string]any{"op": "cancel"})
	case "set-input-mode":
		return call(map[string]any{"op": "set-mode", "mode": action["mode"]})
	case "confirm":
		return call(map[string]any{"op": "commit-reviewed"})
	case "wait-settled", "drain-callbacks":
		if err := d.waitSession(terminal...); err != nil {
			return err
		}
		// Late callbacks get time to (wrongly) act before observation.
		time.Sleep(1500 * time.Millisecond)
		return nil
	}
	return fmt.Errorf("unsupported action %q", op)
}

func (d *driver) scenario(id string, actions []map[string]any) (map[string]any, error) {
	d.trace = nil
	d.netSample = map[string]bool{}
	stop := make(chan struct{})
	sampled := make(chan struct{})
	go func() {
		defer close(sampled)
		for {
			d.sampleNetwork()
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	}()
	var failure error
	for _, action := range actions {
		if err := d.action(action); err != nil {
			failure = fmt.Errorf("%s: %w", action["op"], err)
			break
		}
	}
	close(stop)
	<-sampled
	d.sampleNetwork()
	if failure != nil {
		return nil, failure
	}
	observations, err := d.arionState()
	if err != nil {
		return nil, err
	}
	editor, err := d.editorState()
	if err != nil {
		return nil, err
	}
	observations["editors"] = map[string]any{"A": editor["A"], "B": editor["B"], "secure": editor["secure"]}
	var sockets []string
	for socket := range d.netSample {
		sockets = append(sockets, socket)
	}
	sort.Strings(sockets)
	observations["network"] = map[string]any{"outboundRequests": len(sockets)}
	d.record("network", sockets)
	d.record("observations", observations)
	evidenceDir := d.path(d.config.Evidence)
	if err := os.MkdirAll(evidenceDir, 0o700); err != nil {
		return nil, err
	}
	tracePath := filepath.Join(evidenceDir, fmt.Sprintf("%s-%d.json", id, time.Now().UnixNano()))
	data, err := json.MarshalIndent(d.trace, "", " ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(tracePath, data, 0o600); err != nil {
		return nil, err
	}
	// The digest covers the bytes just written; re-reading the file could
	// only add a failure that would leave the evidence unhashed.
	digest := sha256.Sum256(data)
	relative, _ := filepath.Rel(d.dir, tracePath)
	return map[string]any{"execution": "live", "fixtureOwnership": "isolated", "observations": observations,
		"evidence": []map[string]any{{"path": relative, "sha256": hex.EncodeToString(digest[:])}}}, nil
}

func (d *driver) cleanup() map[string]any {
	if d.arion != nil {
		_, _ = d.arion.call(map[string]any{"op": "quit"})
		d.arion.conn.Close()
	}
	if d.editor != nil {
		_, _ = d.editor.call(map[string]any{"op": "quit"})
		d.editor.conn.Close()
	}
	complete := true
	if d.profile != "" {
		time.Sleep(500 * time.Millisecond)
		complete = os.RemoveAll(d.profile) == nil
	}
	return map[string]any{"execution": "live", "cleanupComplete": complete}
}

func main() {
	configPath := flag.String("config", "", "driver configuration JSON")
	flag.Parse()
	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config unreadable")
		os.Exit(2)
	}
	// Paths sent to the app must be absolute: its working directory is "/".
	dir, err := filepath.Abs(filepath.Dir(*configPath))
	if err != nil {
		os.Exit(2)
	}
	d := &driver{dir: dir}
	if err := json.Unmarshal(data, &d.config); err != nil {
		fmt.Fprintln(os.Stderr, "config invalid")
		os.Exit(2)
	}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 8<<20)
	out := json.NewEncoder(os.Stdout)
	for in.Scan() {
		var req request
		if err := json.Unmarshal(in.Bytes(), &req); err != nil {
			os.Exit(1)
		}
		var reply map[string]any
		switch req.Op {
		case "hello":
			reply, err = d.hello()
		case "scenario":
			reply, err = d.scenario(req.ID, req.Actions)
		case "cleanup":
			reply, err = d.cleanup(), nil
		default:
			err = fmt.Errorf("unsupported op %q", req.Op)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "e2e driver:", err)
			d.cleanup()
			os.Exit(1)
		}
		reply["protocol"], reply["requestId"] = 1, req.RequestID
		if err := out.Encode(reply); err != nil {
			os.Exit(1)
		}
	}
}
