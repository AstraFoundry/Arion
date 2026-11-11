package main

import (
	"strings"
	"testing"
)

func TestBuildRejectsInvalidPlatformsBeforeProjectWork(t *testing.T) {
	for _, target := range []string{"", " , ", "linux", "linux/", "/amd64", "linux/amd64/extra", "freebsd/amd64", "js/amd64", "web,freebsd/amd64"} {
		t.Run(target, func(t *testing.T) {
			err := runBuild([]string{"-platform", target, t.TempDir()})
			if err == nil || !strings.Contains(err.Error(), "platform") {
				t.Fatalf("runBuild(%q) = %v; want a platform error before loading or building the project", target, err)
			}
		})
	}
}

func TestBuildPlatforms(t *testing.T) {
	const raw = " darwin/universal, linux/amd64, windows/arm64, web, js/wasm "
	targets, err := buildPlatforms(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(targets, ","); got != "darwin/universal,linux/amd64,windows/arm64,web,js/wasm" {
		t.Fatalf("targets = %s", got)
	}
}
