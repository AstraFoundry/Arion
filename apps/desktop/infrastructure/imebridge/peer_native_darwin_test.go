//go:build darwin && cgo

package imebridge

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"arion.local/core/domain"
)

// The helper executes the production native handler with a synthetic encrypted
// profile. It does not register an input source or use personal vocabulary.
func TestDarwinSignedNativeVocabulary(t *testing.T) {
	binary := os.Getenv("ARION_SIGNED_NATIVE_PEER")
	if binary == "" {
		t.Skip("requires explicit signed native fixture")
	}
	work := os.Getenv("ARION_SIGNED_PEER_FIXTURES")
	profile := filepath.Join(work, "native-profile")
	if err := os.Mkdir(profile, 0700); err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp("/tmp", "arion-vocab-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	bridge := New(filepath.Join(directory, "bridge.sock"), nil, nil)
	defer bridge.Close()
	bridge.RequireClientVerification(func(peer PeerIdentity, _ ClientHello) error {
		if !peer.BindPublisher(BundleID) {
			return ErrVocabularyRejected
		}
		return nil
	})
	if err = bridge.Listen(); err != nil {
		t.Fatal(err)
	}
	const term = "星槎工坊"
	type probe struct {
		Candidates []string
		Committed  string
	}
	type report struct {
		Passed                 bool
		Error                  string
		NativeLibraryPath      string
		AuthenticatedParentPID int
		InitialRevision        uint64
		InitialProbe           probe
		Exchanges              []struct {
			Request message
			Reply   message
			Probe   probe
		}
	}
	for index, name := range []string{"publish", "restore-delete", "restore-empty"} {
		t.Run(name, func(t *testing.T) {
			output := filepath.Join(work, "evidence", "native-"+name+".json")
			cmd := exec.Command(binary, os.Getenv("ARION_SIGNED_NATIVE_DATA"), profile, filepath.Join(directory, "bridge.sock"), os.Getenv("ARION_SIGNED_PEER_APP_ID"), output)
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				input.Close()
				if cmd.ProcessState == nil {
					cmd.Process.Kill()
					cmd.Wait()
				}
			}()
			deadline := time.Now().Add(15 * time.Second)
			for !bridge.ActiveClient() && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			if !bridge.ActiveClient() {
				t.Fatal("native signed peer did not activate")
			}
			bridge.mu.Lock()
			verified := bridge.clientVerified && bridge.peer.Alive() && bridge.peer.ProcessID == uint32(cmd.Process.Pid)
			bridge.mu.Unlock()
			if !verified {
				t.Fatal("activation is not bound to actual native helper")
			}
			if index < 2 {
				var terms []domain.KeyboardTerm
				if index == 0 {
					terms = []domain.KeyboardTerm{{ID: "synthetic-term", Text: term, Reading: "xing cha gong fang"}}
				}
				if err = bridge.SyncKeyboardVocabulary(context.Background(), terms); err != nil {
					t.Fatal(err)
				}
			}
			input.Close()
			if err = cmd.Wait(); err != nil {
				t.Fatalf("native helper: %v; report: %s", err, output)
			}
			wait(t, func() bool { return !bridge.ActiveClient() })
			data, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			var result report
			if err = json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			library, err := filepath.EvalSymlinks(os.Getenv("ARION_SIGNED_NATIVE_LIBRARY"))
			if err != nil {
				t.Fatal(err)
			}
			if !result.Passed || result.AuthenticatedParentPID != os.Getpid() || result.NativeLibraryPath != library || result.InitialRevision != uint64(index) {
				t.Fatalf("invalid native proof: passed=%v revision=%d error=%s", result.Passed, result.InitialRevision, result.Error)
			}
			if slices.Contains(result.InitialProbe.Candidates, term) != (index == 1) {
				t.Fatal("restart restored incorrect candidate state")
			}
			if index == 2 {
				if len(result.Exchanges) != 0 {
					t.Fatal("restore-only run unexpectedly published")
				}
				return
			}
			if len(result.Exchanges) != 2 {
				t.Fatal("expected query and publication")
			}
			publication := result.Exchanges[1]
			if publication.Request.Op != "vocabulary_publish" || publication.Reply.Status != "ok" || publication.Reply.VocabularyRevision != uint64(index+1) {
				t.Fatal("native publication not acknowledged")
			}
			if slices.Contains(publication.Probe.Candidates, term) != (index == 0) {
				t.Fatal("publication did not change actual candidates")
			}
			if index == 0 && publication.Probe.Committed != term {
				t.Fatal("actual native commit differs from published term")
			}
		})
		if t.Failed() {
			break
		}
	}
}
