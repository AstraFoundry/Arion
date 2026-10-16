package main

import (
	"path/filepath"
	"testing"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/vocabulary"
)

func TestCorrectionRoundTripAndReverse(t *testing.T) {
	dir := t.TempDir()
	r := request{Operation: "init", StorePath: filepath.Join(dir, "vocabulary.enc"), KeyPath: filepath.Join(dir, "key"), UnixTime: 1800000000}
	initial, err := execute(r)
	if err != nil || initial["afterCount"] != 0 {
		t.Fatalf("initial: %+v %v", initial, err)
	}
	r.Operation = "settle"
	r.Edit = domain.FieldEdit{Inserted: "请检查瑞迪斯的设置。", Baseline: "请检查瑞迪斯的设置。", Settled: "请检查Redis的设置。"}
	learned, err := execute(r)
	if err != nil {
		t.Fatal(err)
	}
	entries := learned["entries"].([]vocabulary.Entry)
	if len(entries) != 1 || entries[0].From != "瑞迪斯" || entries[0].To != "Redis" || !entries[0].Approved {
		t.Fatalf("persisted: %+v", entries)
	}
	r.Operation = "snapshot"
	reloaded, err := execute(r)
	if err != nil || reloaded["ciphertextSha256"] != learned["ciphertextSha256"] || reloaded["afterCount"] != 1 {
		t.Fatalf("reload: %+v %v", reloaded, err)
	}
	r.Operation = "settle"
	r.Edit = domain.FieldEdit{Inserted: "请检查Redis的设置。", Baseline: "请检查Redis的设置。", Settled: "请检查瑞迪斯的设置。"}
	reverted, err := execute(r)
	if err != nil || reverted["afterCount"] != 0 {
		t.Fatalf("reverse: %+v %v", reverted, err)
	}
}

func TestContinuedTypingDoesNotCreateMemory(t *testing.T) {
	dir := t.TempDir()
	r := request{Operation: "init", StorePath: filepath.Join(dir, "vocabulary.enc"), KeyPath: filepath.Join(dir, "key"), UnixTime: 1800000000}
	if _, err := execute(r); err != nil {
		t.Fatal(err)
	}
	r.Operation = "settle"
	r.Edit = domain.FieldEdit{Inserted: "材料已经准备好了。", Baseline: "材料已经准备好了。", Settled: "材料已经准备好了。明天再发。"}
	result, err := execute(r)
	if err != nil || result["afterCount"] != 0 {
		t.Fatalf("continued typing: %+v %v", result, err)
	}
}
