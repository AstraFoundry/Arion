//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"fmt"
	wire "github.com/egoist/mygo/internal/notification"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unsafe"
)

var (
	procCoUninitialize       = ole32.NewProc("CoUninitialize")
	procSHGetKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
	procMoveFileExW          = kernel32.NewProc("MoveFileExW")
)

// RegisterNotifications is the installer's short-lived entry point, called
// before application bootstrap or its single-instance lock. Its HKCU identity
// matches the per-user installer; it never changes notification preferences.
func RegisterNotifications(remove bool) error {
	info, ok := (appController{}).Package()
	if !ok {
		return errors.New("mygo: notification registration requires a packaged executable")
	}
	if !validToastAppID(info.Identifier) {
		return errors.New("mygo: invalid notification sender identity")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		return hresultError("CoInitializeEx", hr)
	}
	defer procCoUninitialize.Call()
	clsid := wire.ActivatorCLSID(info.Identifier)
	key := toastRegistryKey(clsid)
	shortcut, err := notificationShortcutPath(info.Name)
	if err != nil {
		return err
	}
	if remove {
		command, exists := regGet(hkeyCurrentUser, key, "")
		if exists && command != toastCommand(exe) {
			return errors.New("mygo: refusing to unregister another executable's notification activator")
		}
		if !exists {
			return nil
		}
		hr, _, _ := procRoInitialize.Call(0)
		if failed(hr) {
			return hresultError("RoInitialize", hr)
		}
		defer procRoUninitialize.Call()
		notifier, history, err := toastInterfaces(info.Identifier)
		if err != nil {
			return err
		}
		defer release(notifier)
		defer release(history)
		if err := removeToast(history, info.Identifier, ""); err != nil {
			return err
		}
		if err := os.Remove(shortcut); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("mygo: remove notification shortcut: %w", err)
		}
		return regDeleteTree(hkeyCurrentUser, `Software\Classes\CLSID\{`+clsid+`}`)
	}
	staged := shortcut + ".mygo-new.lnk"
	defer os.Remove(staged)
	if err := writeNotificationShortcut(staged, exe, info.Identifier, clsid); err != nil {
		return err
	}
	old, existed := regGet(hkeyCurrentUser, key, "")
	if err := regSet(hkeyCurrentUser, key, "", toastCommand(exe)); err != nil {
		return err
	}
	if r, _, callErr := procMoveFileExW.Call(uintptr(unsafe.Pointer(u16(staged))), uintptr(unsafe.Pointer(u16(shortcut))), 1|8); r == 0 {
		var rollback error
		if existed {
			rollback = regSet(hkeyCurrentUser, key, "", old)
		} else {
			rollback = regDeleteTree(hkeyCurrentUser, `Software\Classes\CLSID\{`+clsid+`}`)
		}
		return errors.Join(fmt.Errorf("mygo: promote notification shortcut: %w", callErr), rollback)
	}
	return nil
}
func notificationShortcutPath(name string) (string, error) {
	programs := guid("a77f5d77-2e2b-44c3-a6a2-aba601054a51")
	var folder uintptr
	hr, _, _ := procSHGetKnownFolderPath.Call(uintptr(unsafe.Pointer(&programs)), 0, 0, uintptr(unsafe.Pointer(&folder)))
	if failed(hr) {
		return "", hresultError("SHGetKnownFolderPath Programs", hr)
	}
	defer procCoTaskMemFree.Call(folder)
	return filepath.Join(wstr(folder), notificationShortcutName(name)+".lnk"), nil
}
func writeNotificationShortcut(path, exe, appID, clsid string) error {
	shellLink := guid("00021401-0000-0000-c000-000000000046")
	iidLink := guid("000214f9-0000-0000-c000-000000000046")
	iidStore := guid("886d8eeb-8cf2-4446-8d02-cdba1dbdcf99")
	iidFile := guid("0000010b-0000-0000-c000-000000000046")
	var link uintptr
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&shellLink)), 0, 1, uintptr(unsafe.Pointer(&iidLink)), uintptr(unsafe.Pointer(&link)))
	if failed(hr) {
		return hresultError("create notification shell link", hr)
	}
	defer release(link)
	if hr := comCall(link, 20, uintptr(unsafe.Pointer(u16(exe)))); failed(hr) {
		return hresultError("IShellLink.SetPath", hr)
	}
	if hr := comCall(link, 9, uintptr(unsafe.Pointer(u16(filepath.Dir(exe))))); failed(hr) {
		return hresultError("IShellLink.SetWorkingDirectory", hr)
	}
	if hr := comCall(link, 17, uintptr(unsafe.Pointer(u16(exe))), 0); failed(hr) {
		return hresultError("IShellLink.SetIconLocation", hr)
	}
	store := queryInterface(link, &iidStore)
	if store == 0 {
		return errors.New("mygo: shell link property store unavailable")
	}
	defer release(store)
	type propertyKey struct {
		format GUID
		id     uint32
	}
	type propVariant struct {
		vt       uint16
		reserved [3]uint16
		data     [2]uintptr
	}
	format := guid("9f4c2855-9f79-4b39-a8d0-e1d42de1d5f3")
	idKey := propertyKey{format, 5}
	activatorKey := propertyKey{format, 26}
	text := u16(appID)
	value := propVariant{vt: 31, data: [2]uintptr{uintptr(unsafe.Pointer(text)), 0}} // VT_LPWSTR; SetValue copies.
	if hr := comCall(store, 6, uintptr(unsafe.Pointer(&idKey)), uintptr(unsafe.Pointer(&value))); failed(hr) {
		return hresultError("set shortcut AppUserModelID", hr)
	}
	runtime.KeepAlive(text)
	cls := guid(clsid)
	value = propVariant{vt: 72, data: [2]uintptr{uintptr(unsafe.Pointer(&cls)), 0}} // VT_CLSID
	if hr := comCall(store, 6, uintptr(unsafe.Pointer(&activatorKey)), uintptr(unsafe.Pointer(&value))); failed(hr) {
		return hresultError("set shortcut ToastActivatorCLSID", hr)
	}
	runtime.KeepAlive(cls)
	if hr := comCall(store, 7); failed(hr) {
		return hresultError("commit notification shell link", hr)
	}
	file := queryInterface(link, &iidFile)
	if file == 0 {
		return errors.New("mygo: shell link persistence unavailable")
	}
	defer release(file)
	if hr := comCall(file, 6, uintptr(unsafe.Pointer(u16(path))), 1); failed(hr) {
		return hresultError("save notification shell link", hr)
	}
	return nil
}

func notificationShortcutName(name string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(name))
}
