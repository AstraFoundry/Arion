//go:build !windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"arion.local/desktop/infrastructure/provision"
)

func TestHelloAttestsRelativeAndAbsoluteArtifactPaths(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "engine")
	if err := os.WriteFile(filename, []byte("engine-bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	expected, err := provision.HashFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	config := driverConfig{}
	for _, p := range []string{"engine", filename} {
		config.Artifacts = append(config.Artifacts, struct {
			ID     string `json:"id"`
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		}{ID: p, Path: p, SHA256: expected})
	}
	data, _ := json.Marshal(config)
	configPath := filepath.Join(dir, "driver.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	d := driver{configPath: configPath}
	if _, err := d.hello(request{Purpose: "benchmark"}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"engine", filename} {
		if d.attested[p] != expected {
			t.Fatalf("wrong attestation for %s", p)
		}
	}
}
