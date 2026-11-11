//go:build windows && (amd64 || arm64)

package windows

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

var procSHQueryRecycleBinW = shell32.NewProc("SHQueryRecycleBinW")

// recycled returns how many items the recycle bin of path's drive holds.
func recycled(t *testing.T, path string) int64 {
	t.Helper()
	info := struct {
		Size  uint32
		Bytes int64
		Items int64
	}{}
	info.Size = uint32(unsafe.Sizeof(info))
	root := filepath.VolumeName(path) + `\`
	if r, _, _ := procSHQueryRecycleBinW.Call(uintptr(unsafe.Pointer(u16(root))), uintptr(unsafe.Pointer(&info))); failed(r) {
		t.Fatalf("SHQueryRecycleBin(%s): %#x", root, r)
	}
	return info.Items
}

func TestTrashItem(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	procCoInitializeEx.Call(0, coinitApartmentThreaded)
	dir := t.TempDir()

	file := filepath.Join(dir, "file.txt")
	folder := filepath.Join(dir, "folder")
	os.WriteFile(file, []byte("trash me"), 0o644)
	os.MkdirAll(filepath.Join(folder, "inner"), 0o755)
	os.WriteFile(filepath.Join(folder, "inner", "a.txt"), []byte("a"), 0o644)
	for _, path := range []string{file, folder} {
		before := recycled(t, path)
		if err := (shell{}).TrashItem(path); err != nil {
			t.Fatalf("TrashItem(%s): %v", path, err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s is still there: %v", path, err)
		}
		if after := recycled(t, path); after != before+1 {
			t.Errorf("TrashItem(%s): the recycle bin went from %d to %d items, not to the recycle bin", path, before, after)
		}
	}

	if err := (shell{}).TrashItem(filepath.Join(dir, "missing")); err == nil {
		t.Error("TrashItem of a missing file did not fail")
	}

	// A share has no recycle bin: Windows would delete the file for good,
	// so TrashItem fails and leaves it.
	kept := filepath.Join(dir, "kept.txt")
	os.WriteFile(kept, []byte("keep me"), 0o644)
	vol := filepath.VolumeName(kept)
	unc := `\\localhost\` + strings.TrimSuffix(vol, ":") + `$` + strings.TrimPrefix(kept, vol)
	if _, err := os.Stat(unc); err != nil {
		t.Skipf("no administrative share for %s: %v", vol, err)
	}
	if err := (shell{}).TrashItem(unc); err == nil {
		t.Errorf("TrashItem(%s) did not fail", unc)
	} else {
		t.Log(err)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("TrashItem(%s) deleted the file: %v", unc, err)
	}
}
