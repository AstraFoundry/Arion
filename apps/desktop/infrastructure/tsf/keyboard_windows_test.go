//go:build windows

package tsf

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestKeyboardTreeRejectsTamperingAndIncompleteClosures(t *testing.T) {
	data := []byte("immutable dictionary")
	digest := sha256.Sum256(data)
	identity := keyboardFile{Size: int64(len(data)), SHA256: hex.EncodeToString(digest[:])}
	for _, scenario := range []string{"valid", "changed", "missing", "unlisted", "empty-directory", "traversal", "case-alias", "manifest-alias", "trailing-dot", "trailing-space"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]keyboardFile{"dictionary.bin": identity}
			if err := os.WriteFile(filepath.Join(root, "dictionary.bin"), data, 0600); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "changed":
				if err := os.WriteFile(filepath.Join(root, "dictionary.bin"), []byte("substituted content!"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing":
				files["absent.bin"] = identity
			case "unlisted":
				if err := os.WriteFile(filepath.Join(root, "injected.dll"), data, 0600); err != nil {
					t.Fatal(err)
				}
			case "traversal":
				files["../outside.bin"] = identity
			case "empty-directory":
				if err := os.Mkdir(filepath.Join(root, "unlisted"), 0700); err != nil {
					t.Fatal(err)
				}
			case "trailing-dot":
				files["dictionary.bin."] = identity
			case "trailing-space":
				files["dictionary.bin "] = identity
			case "case-alias":
				files["DICTIONARY.BIN"] = identity
			case "manifest-alias":
				files["RUNTIME.JSON"] = identity
			}
			err := verifyKeyboardFiles(root, files)
			if (err == nil) != (scenario == "valid") {
				t.Fatalf("tree validation: %v", err)
			}
		})
	}
}
