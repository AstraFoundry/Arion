//go:build windows && (amd64 || arm64)

package windows

import (
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	wire "github.com/egoist/mygo/internal/notification"
)

func TestNotificationCOMFactoryAndBoundedActivation(t *testing.T) {
	initToastCOMOnce.Do(initToastCOM)
	b := &Backend{}
	s := &toastState{appID: "si.astrafoundry.arion.notification-probe"}
	s.activator = newToastCOMObject(&toastActivationVtbl[0], s, b, false)
	s.factory = newToastCOMObject(&toastFactoryVtbl[0], s, b, true)
	s.factory.target = s.activator
	addRef(uintptr(unsafe.Pointer(s.activator)))
	defer release(uintptr(unsafe.Pointer(s.factory)))
	defer release(uintptr(unsafe.Pointer(s.activator)))
	var activation uintptr
	hr := comCall(uintptr(unsafe.Pointer(s.factory)), 3, 0, uintptr(unsafe.Pointer(&iidNotificationActivation)), uintptr(unsafe.Pointer(&activation)))
	if failed(hr) || activation == 0 {
		t.Fatalf("CreateInstance: %#x", hr)
	}
	defer release(activation)
	unknown := guid("deadbeef-1111-2222-3333-444444444444")
	if got := queryInterface(activation, &unknown); got != 0 {
		release(got)
		t.Fatal("claimed an unknown interface")
	}
	app := u16(s.appID)
	args := u16(wire.Activation("stable/1.2.3/available", "later"))
	for i := 0; i < 32; i++ {
		if hr := comCall(activation, 3, uintptr(unsafe.Pointer(app)), uintptr(unsafe.Pointer(args)), 0, 0); failed(hr) {
			t.Fatalf("Activate %d: %#x", i, hr)
		}
	}
	if hr := comCall(activation, 3, uintptr(unsafe.Pointer(app)), uintptr(unsafe.Pointer(args)), 0, 0); !failed(hr) {
		t.Fatal("unbounded action queue")
	}
	if len(s.pending) != 32 || s.pending[0].action != "later" {
		t.Fatalf("pending=%+v", s.pending)
	}
	s.pending = nil
	bad := u16("https://untrusted.invalid/execute")
	if hr := comCall(activation, 3, uintptr(unsafe.Pointer(app)), uintptr(unsafe.Pointer(bad)), 0, 0); !failed(hr) {
		t.Fatal("accepted raw URL activation")
	}
	runtime.KeepAlive(app)
	runtime.KeepAlive(args)
	runtime.KeepAlive(bad)
}

func TestNotificationShortcutIdentityProperties(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		t.Fatal(hresultError("CoInitializeEx", hr))
	}
	defer procCoUninitialize.Call()
	path := filepath.Join(t.TempDir(), "Probe With Spaces.lnk")
	exe := `C:\Test Path With Spaces\Notification Probe.exe`
	appID := "si.astrafoundry.arion.notification-probe"
	clsid := wire.ActivatorCLSID(appID)
	if err := writeNotificationShortcut(path, exe, appID, clsid); err != nil {
		t.Fatal(err)
	}
	clsLink := guid("00021401-0000-0000-c000-000000000046")
	iidLink := guid("000214f9-0000-0000-c000-000000000046")
	iidFile := guid("0000010b-0000-0000-c000-000000000046")
	iidStore := guid("886d8eeb-8cf2-4446-8d02-cdba1dbdcf99")
	var link uintptr
	hr, _, _ = procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsLink)), 0, 1, uintptr(unsafe.Pointer(&iidLink)), uintptr(unsafe.Pointer(&link)))
	if failed(hr) {
		t.Fatal(hresultError("CoCreateInstance", hr))
	}
	defer release(link)
	file := queryInterface(link, &iidFile)
	if file == 0 {
		t.Fatal("IPersistFile unavailable")
	}
	defer release(file)
	if hr := comCall(file, 5, uintptr(unsafe.Pointer(u16(path))), 0); failed(hr) {
		t.Fatal(hresultError("load shortcut", hr))
	}
	store := queryInterface(link, &iidStore)
	if store == 0 {
		t.Fatal("IPropertyStore unavailable")
	}
	defer release(store)
	type key struct {
		format GUID
		id     uint32
	}
	type variant struct {
		vt       uint16
		reserved [3]uint16
		data     [2]uintptr
	}
	format := guid("9f4c2855-9f79-4b39-a8d0-e1d42de1d5f3")
	var id, activator variant
	idKey, clsKey := key{format, 5}, key{format, 26}
	clear := ole32.NewProc("PropVariantClear")
	if hr := comCall(store, 5, uintptr(unsafe.Pointer(&idKey)), uintptr(unsafe.Pointer(&id))); failed(hr) {
		t.Fatal(hresultError("read AppUserModelID", hr))
	}
	defer clear.Call(uintptr(unsafe.Pointer(&id)))
	if hr := comCall(store, 5, uintptr(unsafe.Pointer(&clsKey)), uintptr(unsafe.Pointer(&activator))); failed(hr) {
		t.Fatal(hresultError("read ToastActivatorCLSID", hr))
	}
	defer clear.Call(uintptr(unsafe.Pointer(&activator)))
	if id.vt != 31 || wstr(id.data[0]) != appID {
		t.Fatalf("AppUserModelID variant %+v", id)
	}
	if activator.vt != 72 || *(*GUID)(native(activator.data[0])) != guid(clsid) {
		t.Fatalf("activator variant %+v", activator)
	}
}
