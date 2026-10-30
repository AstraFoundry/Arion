package keyboardmodel

import (
	"arion.local/desktop/infrastructure/rewriting"
	"arion.local/models"
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func request() contracts.KeyboardRankRequest {
	return contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "test", Generation: 1}, Revision: 1, Input: "shijie", Before: "欢迎来到这个", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "a", Text: "时节", Source: "sentence"}, End: 6},
		{Candidate: domain.Candidate{ID: "b", Text: "世界", Source: "sentence"}, End: 6},
	}}
}
func TestWorkerProcess(t *testing.T) {
	mode := os.Getenv("ARION_TEST_KEYBOARD_WORKER")
	if mode == "" {
		return
	}
	fmt.Println(`{"ready":true}`)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if mode == "hang" {
			time.Sleep(10 * time.Second)
		}
		fmt.Println(`{"pick":"b"}`)
	}
	os.Exit(0)
}
func TestWorkerWatchdog(t *testing.T) {
	t.Setenv("ARION_TEST_KEYBOARD_WORKER", "hang")
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	worker, err := Open(ctx, binary, "-test.run=^TestWorkerProcess$")
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	start := time.Now()
	if _, err = worker.RankKeyboard(ctx, request()); err == nil {
		t.Fatal("hung worker accepted")
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("worker was not retired within watchdog bound")
	}
}
func TestWorkerRealModel(t *testing.T) {
	model, binary := os.Getenv("ARION_TEST_KEYBOARD_MODEL"), os.Getenv("ARION_TEST_KEYBOARD_BINARY")
	if model == "" || binary == "" {
		t.Skip("set ARION_TEST_KEYBOARD_MODEL and ARION_TEST_KEYBOARD_BINARY for real inference")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	worker, err := Open(ctx, binary, model)
	if err != nil {
		t.Fatal(err)
	}
	defer worker.Close()
	req := request()
	start := time.Now()
	out, err := worker.RankKeyboard(ctx, req)
	if err != nil || out.Candidates[0].ID != "b" {
		t.Fatalf("real inference: %+v %v", out, err)
	}
	t.Logf("real model inference: %s", time.Since(start))
	req.Candidates[0].Source = "phrase"
	out, err = worker.RankKeyboard(ctx, req)
	if err != nil || out.Candidates[0].ID != "a" {
		t.Fatalf("trusted dictionary leader must remain: %+v %v", out, err)
	}
	req.Before = strings.Repeat("x", 2049)
	if _, err = worker.RankKeyboard(ctx, req); err == nil {
		t.Fatal("oversized context accepted")
	}
	req = request()
	req.SessionID = "second-editor"
	if _, err = worker.RankKeyboard(ctx, req); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRealBundledModel(t *testing.T) {
	model, binary := os.Getenv("ARION_TEST_KEYBOARD_MODEL"), os.Getenv("ARION_TEST_KEYBOARD_BINARY")
	if model == "" || binary == "" {
		t.Skip("explicit real model and worker required")
	}
	catalog, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	bundle := t.TempDir()
	executable := filepath.Join(bundle, filepath.Base(binary))
	bytes, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(executable, bytes, 0700); err != nil {
		t.Fatal(err)
	}
	service := New(catalog, executable)
	defer service.Disable()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = service.Enable(ctx); err == nil || service.Status().Installed {
		t.Fatal("missing bundled weights accepted")
	}
	bytes, err = os.ReadFile(model)
	if err != nil {
		t.Fatal(err)
	}
	weights := filepath.Join(bundle, "sentence-model.safetensors")
	if err = os.WriteFile(weights, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	if err = service.Enable(ctx); err != nil {
		t.Fatal(err)
	}
	if service.Status().State != rewriting.Ready {
		t.Fatal(service.Status())
	}
	if out, err := service.RankKeyboard(ctx, request()); err != nil || out.Candidates[0].ID != "b" {
		t.Fatal(out, err)
	}
	service.SuspendKeyboard(true)
	if _, err = service.RankKeyboard(ctx, request()); domain.CodeOf(err) != domain.CodeModelMissing {
		t.Fatal("speech did not suspend fast worker", err)
	}
	service.Disable()
	bytes[0] ^= 1
	if err = os.WriteFile(weights, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	if err = service.Enable(ctx); err == nil {
		t.Fatal("tampered bundled weights accepted")
	}
}
