//go:build linux && (amd64 || arm64)

package gamepad

import (
	"encoding/binary"
	"os"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/internal/wait"
)

// uinput makes virtual input devices.
const (
	uiDevCreate  = 1<<0 | 'U'<<8 | 1 // _IO('U', 1)
	uiDevDestroy = 'U'<<8 | 2
	uiSetEvBit   = 1<<30 | 4<<16 | 'U'<<8 | 100
	uiSetKeyBit  = 1<<30 | 4<<16 | 'U'<<8 | 101
	uiSetAbsBit  = 1<<30 | 4<<16 | 'U'<<8 | 103
)

type uinputSetup struct {
	id   inputID
	name [80]byte
	ff   uint32
}

type uinputAbsSetup struct {
	code uint16
	_    uint16
	info absInfo
}

func uiDevSetup() uintptr { return 1<<30 | unsafe.Sizeof(uinputSetup{})<<16 | 'U'<<8 | 3 }
func uiAbsSetup() uintptr { return 1<<30 | unsafe.Sizeof(uinputAbsSetup{})<<16 | 'U'<<8 | 4 }

// A virtual pad, made with uinput, reads as the kernel's drivers report
// pads: its buttons by place, sticks, triggers and hat. It needs
// /dev/uinput and /dev/input, as in the Linux test container
// (MYGO_UINPUT=1).
func TestEvdev(t *testing.T) {
	if os.Getenv("MYGO_UINPUT") == "" {
		t.Skip("MYGO_UINPUT not set")
	}
	// A DualSense reports its left button, Square, as BTN_WEST; an Xbox
	// controller its left one, X, as BTN_X, BTN_NORTH's code.
	t.Run("PlayStation", func(t *testing.T) { testEvdev(t, 0x054c, btnWest) })
	t.Run("Xbox", func(t *testing.T) { testEvdev(t, 0x045e, btnNorth) })
}

func testEvdev(t *testing.T, vendor uint16, west uint16) {
	fd, err := syscall.Open("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		t.Skip(err)
	}
	defer syscall.Close(fd)
	set := func(req uintptr, v int) {
		if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(v)); e != 0 {
			t.Fatalf("ioctl %#x %d: %v", req, v, e)
		}
	}
	set(uiSetEvBit, evKey)
	set(uiSetEvBit, evAbs)
	for _, k := range []int{btnSouth, btnEast, btnNorth, btnWest, btnTL, btnTR, btnSelect, btnStart, btnMode, btnThumbL, btnThumbR} {
		set(uiSetKeyBit, k)
	}
	axes := map[uint16]absInfo{
		absX: {minimum: -32768, maximum: 32767}, absY: {minimum: -32768, maximum: 32767},
		absRX: {minimum: -32768, maximum: 32767}, absRY: {minimum: -32768, maximum: 32767},
		absZ: {maximum: 255}, absRZ: {maximum: 255},
		absHat0X: {minimum: -1, maximum: 1}, absHat0Y: {minimum: -1, maximum: 1},
	}
	for code, info := range axes {
		set(uiSetAbsBit, int(code))
		s := uinputAbsSetup{code: code, info: info}
		if !ioctl(fd, uiAbsSetup(), unsafe.Pointer(&s)) {
			t.Fatal("UI_ABS_SETUP")
		}
	}
	setup := uinputSetup{id: inputID{bustype: 3, vendor: vendor, product: 0x0ce6}}
	copy(setup.name[:], "Virtual Pad")
	if !ioctl(fd, uiDevSetup(), unsafe.Pointer(&setup)) {
		t.Fatal("UI_DEV_SETUP")
	}
	set(uiDevCreate, 0)
	// Destroyed by the test, unless it fails first.
	defer syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uiDevDestroy, 0)

	var pad *Pad
	wait.Until(t, "the virtual pad", func() bool {
		scanned = time.Time{}
		for _, p := range Pads() {
			if p.Name() == "Virtual Pad" {
				pad = p
			}
		}
		return pad != nil
	})
	emit := func(typ, code uint16, value int32) {
		var e [24]byte
		binary.LittleEndian.PutUint16(e[16:], typ)
		binary.LittleEndian.PutUint16(e[18:], code)
		binary.LittleEndian.PutUint32(e[20:], uint32(value))
		if _, err := syscall.Write(fd, e[:]); err != nil {
			t.Fatal(err)
		}
	}
	emit(evKey, btnSouth, 1)
	emit(evKey, west, 1)
	emit(evAbs, absX, 32767)
	emit(evAbs, absRY, -32768)
	emit(evAbs, absRZ, 255)
	emit(evAbs, absHat0Y, -1)
	emit(0, 0, 0) // SYN_REPORT
	// State drains what the kernel passed on so far: wait until it has.
	wait.Until(t, "the report read", func() bool { return pad.State().Axes[LeftX] >= 0.99 })
	s := pad.State()
	for _, b := range []Button{South, West, RightTrigger, Up} {
		if !s.Pressed(b) {
			t.Errorf("button %d not pressed: %v", b, s.Buttons)
		}
	}
	for _, b := range []Button{East, North, LeftTrigger, Down} {
		if s.Pressed(b) {
			t.Errorf("button %d pressed: %v", b, s.Buttons)
		}
	}
	if s.Axes[LeftX] < 0.99 || s.Axes[RightY] > -0.99 || s.Axes[LeftY] < -0.01 || s.Axes[LeftY] > 0.01 {
		t.Errorf("axes %v", s.Axes)
	}
	emit(evKey, btnSouth, 0)
	emit(0, 0, 0)
	wait.Until(t, "South released", func() bool { return !pad.State().Pressed(South) })
	set(uiDevDestroy, 0)
	wait.Until(t, "the pad disconnected once destroyed", func() bool { pad.State(); return !pad.Connected() })
}
