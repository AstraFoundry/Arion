// Command updaterfixture is an isolated native lifecycle probe for the staged
// updater E2E test. It only updates the temporary bundle holding this binary.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
)

type observation struct {
	Version  string `json:"version"`
	Resource string `json:"resource"`
	Stage    string `json:"stage,omitempty"`
	Error    string `json:"error,omitempty"`
}

func main() {
	state := os.Getenv("MYGO_UPDATER_FIXTURE_DIR")
	if state == "" {
		panic("updater fixture requires an isolated state directory")
	}
	mygo.App.SetName("Arion updater fixture")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	mygo.App.WhenReady(func() { go run(state) })
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(state string) {
	result := observation{Version: mygo.App.Version()}
	exe, err := os.Executable()
	if err == nil {
		var b []byte
		b, err = os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(exe)), "Resources", "version"))
		result.Resource = string(b)
	}
	write := func(event string, err error) {
		if err != nil {
			result.Error = err.Error()
		}
		b, marshalErr := json.Marshal(result)
		if marshalErr == nil {
			marshalErr = os.WriteFile(filepath.Join(state, event+".json"), b, 0o600)
		}
		if marshalErr != nil {
			fmt.Fprintln(os.Stderr, marshalErr)
		}
	}
	if err != nil {
		write("failure", err)
		mygo.App.Quit()
		return
	}
	if result.Version == "2.0.0" {
		b, err := os.ReadFile(filepath.Join(state, "prepared.json"))
		var saved observation
		if err == nil {
			err = json.Unmarshal(b, &saved)
		}
		if err != nil {
			write("failure", err)
			mygo.App.Quit()
			return
		}
		receipt, err := mygo.Updater.RecoverPrepared(saved.Stage)
		if err == nil && !receipt.Applied() {
			err = errors.New("completed fixture receipt was not recognized")
		}
		if err == nil {
			err = os.Remove(filepath.Join(state, "prepared.json"))
		}
		if err == nil {
			err = receipt.Acknowledge()
		}
		if err != nil {
			write("failure", err)
		} else {
			write("relaunched", nil)
		}
		mygo.App.Quit()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	switch os.Getenv("MYGO_UPDATER_FIXTURE_ACTION") {
	case "prepare", "bad-signature":
		found, checkErr := mygo.Updater.Check(ctx)
		if checkErr != nil {
			write("failure", checkErr)
			break
		}
		if found == nil {
			write("failure", errors.New("no fixture update"))
			break
		}
		prepared, prepareErr := found.Prepare(ctx, nil)
		if os.Getenv("MYGO_UPDATER_FIXTURE_ACTION") == "bad-signature" {
			if prepareErr == nil {
				prepared.Discard()
				write("failure", errors.New("accepted invalid fixture signature"))
			} else {
				write("rejected", prepareErr)
			}
			break
		}
		if prepareErr != nil {
			write("failure", prepareErr)
			break
		}
		result.Stage = prepared.Path()
		write("prepared", nil)
	case "apply":
		b, readErr := os.ReadFile(filepath.Join(state, "prepared.json"))
		var saved observation
		if readErr == nil {
			readErr = json.Unmarshal(b, &saved)
		}
		if readErr != nil {
			write("failure", readErr)
			break
		}
		prepared, recoverErr := mygo.Updater.RecoverPrepared(saved.Stage)
		if recoverErr != nil {
			write("failure", recoverErr)
			break
		}
		if applyErr := prepared.Apply(); applyErr != nil {
			write("failure", applyErr)
			break
		}
		write("applied", nil)
		mygo.App.Relaunch()
		return
	default:
		write("failure", errors.New("unknown fixture action"))
	}
	mygo.App.Quit()
}
