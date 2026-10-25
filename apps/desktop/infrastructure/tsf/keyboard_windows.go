//go:build windows

package tsf

import (
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"arion.local/core/domain"
)

type keyboardFile struct {
	Size   int64  `json:"sizeBytes"`
	SHA256 string `json:"sha256"`
}

func validateKeyboardRuntime(root, expected string) error {
	fail := func(detail string) error {
		return domain.Fail(domain.CodeArtifactUnverified, "Windows keyboard "+detail)
	}
	manifest, err := os.ReadFile(filepath.Join(root, "runtime.json"))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(manifest)
	if len(expected) != 64 || hex.EncodeToString(digest[:]) != expected {
		return fail("manifest identity differs")
	}
	var metadata struct {
		Schema  string                  `json:"schema"`
		Library string                  `json:"library"`
		Files   map[string]keyboardFile `json:"files"`
	}
	if json.Unmarshal(manifest, &metadata) != nil || metadata.Schema != "arion_pinyin" || metadata.Library != "lib/ArionRime.dll" {
		return fail("manifest is invalid")
	}
	for _, member := range []string{"lib/ArionKeyboard.dll", "lib/ArionRime.dll", "data/build/default.yaml", "data/build/arion_pinyin.schema.yaml", "data/build/pinyin_simp.table.bin", "data/build/pinyin_simp.prism.bin", "data/build/pinyin_simp.reverse.bin"} {
		if _, found := metadata.Files[member]; !found {
			return fail("required file is missing: " + member)
		}
	}
	if err := verifyKeyboardFiles(root, metadata.Files); err != nil {
		return fail(err.Error())
	}
	for _, name := range []string{"ArionKeyboard.dll", "ArionRime.dll"} {
		image, err := pe.Open(filepath.Join(root, "lib", name))
		if err != nil {
			return fail("runtime binary is invalid")
		}
		machine := image.Machine
		image.Close()
		if machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			return fail("runtime binary is not x64")
		}
	}
	return nil
}

// The manifest defines a closed immutable tree. Reject path aliases, links,
// unlisted files and mismatched bytes before the administrator transaction.
func verifyKeyboardFiles(root string, files map[string]keyboardFile) error {
	var err error
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	seen := make(map[string]bool, len(files))
	directories := map[string]bool{".": true}
	for member, record := range files {
		if strings.EqualFold(member, "runtime.json") || !fs.ValidPath(member) || strings.ContainsAny(member, "\\:\x00\"<>|?*") {
			return fmt.Errorf("unsafe member %q", member)
		}
		for _, part := range strings.Split(member, "/") {
			if strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || strings.ContainsFunc(part, func(r rune) bool { return r < 32 }) {
				return fmt.Errorf("unsafe member %q", member)
			}
		}
		for parent := filepath.Dir(filepath.FromSlash(member)); parent != "."; parent = filepath.Dir(parent) {
			directories[filepath.ToSlash(parent)] = true
		}
		folded := strings.ToLower(member)
		if seen[folded] {
			return fmt.Errorf("duplicate member %q", member)
		}
		seen[folded] = true
		if record.Size <= 0 || len(record.SHA256) != 64 {
			return fmt.Errorf("invalid identity for %q", member)
		}
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("linked runtime member")
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(absolute)) {
			return fmt.Errorf("redirected runtime member")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		member := filepath.ToSlash(relative)
		if entry.IsDir() {
			if !directories[member] {
				return fmt.Errorf("unlisted directory %q", member)
			}
			return nil
		}
		if member == "runtime.json" {
			return nil
		}
		record, found := files[member]
		if !found {
			return fmt.Errorf("unlisted member %q", member)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() != record.Size {
			return fmt.Errorf("invalid file %q", member)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != record.SHA256 {
			return fmt.Errorf("hash differs for %q", member)
		}
		delete(seen, strings.ToLower(member))
		return nil
	})
	if err != nil {
		return err
	}
	if len(seen) != 0 {
		return fmt.Errorf("manifest contains missing files")
	}
	return nil
}
