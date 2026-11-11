//go:build windows

package d3d11

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testEffect = "float4 effect(float2 p, Effect e) { return float4(e.p0.rgb, 1); }\n"

// TestEffectCache checks that an effect's code compiles once for all its
// users, as CompileEffect compiles it, and that a later launch reads it
// from the disk.
func TestEffectCache(t *testing.T) {
	want, err := CompileEffect(testEffect)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	c := &effectCache{byKey: map[string]*effectEntry{}, dir: dir}
	got, err := c.code(testEffect)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("code: %d bytes, %v; want %d bytes", len(got), err, len(want))
	}
	if again, _ := c.code(testEffect); &again[0] != &got[0] {
		t.Error("the code compiled again")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.dxbc"))
	if len(files) != 1 {
		t.Fatalf("files kept: %v", files)
	}
	// A file that is not whole compiles again.
	os.WriteFile(files[0], want[:len(want)/2], 0o644)
	if got, _ := (&effectCache{byKey: map[string]*effectEntry{}, dir: dir}).code(testEffect); !bytes.Equal(got, want) {
		t.Error("a broken file was read")
	}
	// The next launch reads the file.
	marked := append([]byte(nil), want...)
	marked[len(marked)-1] ^= 1
	os.WriteFile(files[0], marked, 0o644)
	if got, _ := (&effectCache{byKey: map[string]*effectEntry{}, dir: dir}).code(testEffect); !bytes.Equal(got, marked) {
		t.Error("the file was not read")
	}
}

// TestTrimEffectCache checks that the cache removes the files least
// recently used beyond its size.
func TestTrimEffectCache(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, effectCacheSize/2)
	for i, name := range []string{"a", "b", "c"} {
		path := filepath.Join(dir, name)
		os.WriteFile(path, big, 0o644)
		at := time.Unix(int64(1000*(3-i)), 0) // a is the newest
		os.Chtimes(path, at, at)
	}
	trimEffectCache(dir)
	for name, kept := range map[string]bool{"a": true, "b": true, "c": false} {
		if _, err := os.Stat(filepath.Join(dir, name)); (err == nil) != kept {
			t.Errorf("%s: kept %v, want %v", name, err == nil, kept)
		}
	}
}
