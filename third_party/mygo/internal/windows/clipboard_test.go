//go:build windows && (amd64 || arm64)

package windows

import (
	"encoding/binary"
	"image"
	"runtime"
	"slices"
	"testing"

	"github.com/egoist/mygo/transfer"
)

func TestClipboardDIBRejectsInvalidGeometry(t *testing.T) {
	for _, dimensions := range [][2]uint32{{0, 1}, {1, 0}, {0x7fffffff, 0x7fffffff}, {2, 0x80000000}, {1 << 20, 1 << 20}} {
		dib := make([]byte, 40)
		binary.LittleEndian.PutUint32(dib, 40)
		binary.LittleEndian.PutUint32(dib[4:], dimensions[0])
		binary.LittleEndian.PutUint32(dib[8:], dimensions[1])
		binary.LittleEndian.PutUint16(dib[12:], 1)
		binary.LittleEndian.PutUint16(dib[14:], 32)
		if img := dibToImage(dib); img != nil {
			t.Fatalf("malformed dimensions accepted: %v", dimensions)
		}
	}
	dib := imageToDIB(image.NewNRGBA(image.Rect(0, 0, 2, 3)))
	if img := dibToImage(dib); img == nil || img.Bounds() != image.Rect(0, 0, 2, 3) {
		t.Fatal("valid top-down DIB rejected")
	}
	binary.LittleEndian.PutUint32(dib, 0)
	if dibToImage(dib) != nil {
		t.Fatal("invalid bitmap header accepted")
	}
}

// A representation offered but unreadable, as the EnterpriseDataProtectionId
// Notepad offers with no data behind it, costs a read of everything offered
// nothing more than itself: the text still reads (egoist/kero#286).
func TestClipboardReadDataSkipsUnreadable(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// A window of a predefined class owns the clipboard and renders nothing
	// when asked for the representation it only promised.
	hwnd := createWindow(0, "STATIC", "", 0, 0, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("cannot create the clipboard's owner")
	}
	defer procDestroyWindow.Call(hwnd)
	c := clipboard{&Backend{appHwnd: hwnd}}
	if !c.open() {
		t.Fatal("cannot open the clipboard")
	}
	procEmptyClipboard.Call()
	if !setClipboardData(cfUnicodeText, utf16Bytes("KERO_TEST_123")) {
		closeClipboard()
		t.Fatal("cannot put text on the clipboard")
	}
	// Promised, for the owner to render later (delayed rendering), which
	// it never does. Formats below tells whether the promise was taken.
	id := registerClipboardFormat("EnterpriseDataProtectionId")
	procSetClipboardData.Call(uintptr(id), 0)
	closeClipboard()
	defer func() {
		if c.open() {
			procEmptyClipboard.Call()
			closeClipboard()
		}
	}()
	if !slices.Contains(c.Formats(), "EnterpriseDataProtectionId") {
		t.Fatalf("the promised representation is not offered: %q", c.Formats())
	}
	d, err := c.ReadData(nil)
	if err != nil {
		t.Fatalf("reading everything offered: %v", err)
	}
	if text, _ := d.Read(transfer.Text); string(text) != "KERO_TEST_123" {
		t.Fatalf("text %q, formats %q", text, d.Formats())
	}
	// Asked for by name, the unreadable representation is still an error.
	if _, err := c.ReadData([]transfer.Format{"EnterpriseDataProtectionId"}); err == nil {
		t.Fatal("reading the unreadable representation by name succeeded")
	}
}
