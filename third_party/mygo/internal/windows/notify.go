//go:build windows && (amd64 || arm64)

package windows

import (
	"errors"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	wire "github.com/egoist/mygo/internal/notification"
	"github.com/egoist/mygo/internal/platform"
)

// ABI: Windows SDK metadata, also published by microsoft/windows-rs under
// crates/libs/windows/src/Windows/UI/Notifications. This uses OS WinRT rather
// than the separately deployed Windows App SDK or notification-area balloons.
var (
	combase                                     = systemDLL("combase.dll")
	procRoInitialize                            = combase.NewProc("RoInitialize")
	procRoUninitialize                          = combase.NewProc("RoUninitialize")
	procRoGetActivationFactory                  = combase.NewProc("RoGetActivationFactory")
	procRoActivateInstance                      = combase.NewProc("RoActivateInstance")
	procWindowsCreateString                     = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString                     = combase.NewProc("WindowsDeleteString")
	procCoRegisterClassObject                   = ole32.NewProc("CoRegisterClassObject")
	procCoRevokeClassObject                     = ole32.NewProc("CoRevokeClassObject")
	procSetCurrentProcessExplicitAppUserModelID = shell32.NewProc("SetCurrentProcessExplicitAppUserModelID")
	iidToastManager                             = guid("50ac103f-d235-4598-bbef-98fe4d1a3ad4")
	iidToastManager2                            = guid("7ab93c52-0e48-4750-ba9d-1a4113981847")
	iidToastFactory                             = guid("04124b20-82c6-4229-b109-fd9ed4662b53")
	iidToast2                                   = guid("9dfb9fd1-143a-490e-90bf-b9fba7132de7")
	iidXMLDocumentIO                            = guid("6cd0e74e-ee65-4489-9ebf-ca43e87ba637")
	iidNotificationActivation                   = guid("53e31837-6600-4a81-9395-75cffe746f94")
	iidClassFactory                             = guid("00000001-0000-0000-c000-000000000046")
	iidUnknown                                  = guid("00000000-0000-0000-c000-000000000046")
)

const toastManagerClass = "Windows.UI.Notifications.ToastNotificationManager"
const toastGroup = "mygo"

type toastState struct {
	appID              string
	notifier, history  uintptr
	cookie             uint32
	initialized        bool
	factory, activator *toastCOMObject
	err                error
	mu                 sync.Mutex
	pending            []toastActivation
	closed             bool
	relay              func(id, action string) error
}
type toastActivation struct{ id, action string }

func hstring(s string) (uintptr, error) {
	units := utf16.Encode([]rune(s))
	var data *uint16
	if len(units) > 0 {
		data = &units[0]
	}
	var h uintptr
	hr, _, _ := procWindowsCreateString.Call(uintptr(unsafe.Pointer(data)), uintptr(len(units)), uintptr(unsafe.Pointer(&h)))
	runtime.KeepAlive(units)
	if failed(hr) {
		return 0, hresultError("WindowsCreateString", hr)
	}
	return h, nil
}
func activationFactory(name string, iid *GUID) (uintptr, error) {
	h, err := hstring(name)
	if err != nil {
		return 0, err
	}
	defer procWindowsDeleteString.Call(h)
	var out uintptr
	hr, _, _ := procRoGetActivationFactory.Call(h, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	if failed(hr) {
		return 0, hresultError("RoGetActivationFactory "+name, hr)
	}
	return out, nil
}
func activateRuntime(name string) (uintptr, error) {
	h, err := hstring(name)
	if err != nil {
		return 0, err
	}
	defer procWindowsDeleteString.Call(h)
	var out uintptr
	hr, _, _ := procRoActivateInstance.Call(h, uintptr(unsafe.Pointer(&out)))
	if failed(hr) {
		return 0, hresultError("RoActivateInstance "+name, hr)
	}
	return out, nil
}
func toastInterfaces(appID string) (uintptr, uintptr, error) {
	manager, err := activationFactory(toastManagerClass, &iidToastManager)
	if err != nil {
		return 0, 0, err
	}
	defer release(manager)
	app, err := hstring(appID)
	if err != nil {
		return 0, 0, err
	}
	defer procWindowsDeleteString.Call(app)
	var notifier, history uintptr
	if hr := comCall(manager, 7, app, uintptr(unsafe.Pointer(&notifier))); failed(hr) {
		return 0, 0, hresultError("CreateToastNotifierWithId", hr)
	}
	manager2 := queryInterface(manager, &iidToastManager2)
	if manager2 == 0 {
		release(notifier)
		return 0, 0, errors.New("mygo: toast history is unavailable")
	}
	defer release(manager2)
	if hr := comCall(manager2, 6, uintptr(unsafe.Pointer(&history))); failed(hr) {
		release(notifier)
		return 0, 0, hresultError("ToastNotificationManager.History", hr)
	}
	return notifier, history, nil
}
func (b *Backend) initNotifications() {
	s := &toastState{}
	b.toast = s
	info, packaged := appController{b}.Package()
	if !packaged {
		s.err = platform.ErrUnsupported
		return
	}
	s.appID = info.Identifier
	if !validToastAppID(s.appID) {
		s.err = errors.New("mygo: invalid notification sender identity")
		return
	}
	exe, err := os.Executable()
	if err != nil {
		s.err = err
		return
	}
	clsid := wire.ActivatorCLSID(s.appID)
	command, ok := regGet(hkeyCurrentUser, toastRegistryKey(clsid), "")
	if !ok || command != toastCommand(exe) {
		s.err = errors.New("mygo: native toast sender is not registered for this executable; reinstall the application")
		return
	}
	if !has(procRoInitialize) {
		s.err = platform.ErrUnsupported
		return
	}
	hr, _, _ := procRoInitialize.Call(0) // Same STA as UI.
	if failed(hr) {
		s.err = hresultError("RoInitialize", hr)
		return
	}
	s.initialized = true
	s.notifier, s.history, s.err = toastInterfaces(s.appID)
	if s.err != nil {
		b.closeNotifications()
		return
	}
	hr, _, _ = procSetCurrentProcessExplicitAppUserModelID.Call(uintptr(unsafe.Pointer(u16(s.appID))))
	if failed(hr) {
		s.err = hresultError("SetCurrentProcessExplicitAppUserModelID", hr)
		b.closeNotifications()
		return
	}
	initToastCOMOnce.Do(initToastCOM)
	s.activator = newToastCOMObject(&toastActivationVtbl[0], s, b, false)
	s.factory = newToastCOMObject(&toastFactoryVtbl[0], s, b, true)
	s.factory.target = s.activator
	addRef(uintptr(unsafe.Pointer(s.activator)))
	cls := guid(clsid)
	hr, _, _ = procCoRegisterClassObject.Call(uintptr(unsafe.Pointer(&cls)), uintptr(unsafe.Pointer(s.factory)), 4, 1, uintptr(unsafe.Pointer(&s.cookie))) // LOCAL_SERVER, MULTIPLEUSE
	if failed(hr) {
		s.err = hresultError("CoRegisterClassObject", hr)
		b.closeNotifications()
	}
}
func (b *Backend) closeNotifications() {
	s := b.toast
	if s == nil {
		return
	}
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if s.cookie != 0 {
		procCoRevokeClassObject.Call(uintptr(s.cookie))
		s.cookie = 0
	}
	release(s.notifier)
	release(s.history)
	s.notifier, s.history = 0, 0
	if s.initialized {
		procRoUninitialize.Call()
		s.initialized = false
	}
	if s.factory != nil {
		release(uintptr(unsafe.Pointer(s.factory)))
		s.factory = nil
	}
	if s.activator != nil {
		release(uintptr(unsafe.Pointer(s.activator)))
		s.activator = nil
	}
}
func (b *Backend) NotificationsSupported() bool {
	return b.toast != nil && b.toast.err == nil && b.toast.notifier != 0
}
func (b *Backend) NotificationPermission(done func(platform.NotificationAuthorization, error)) {
	if !b.NotificationsSupported() {
		err := platform.ErrUnsupported
		if b.toast != nil && b.toast.err != nil {
			err = b.toast.err
		}
		done(platform.NotificationAuthorizationUnsupported, err)
		return
	}
	var setting int32
	if hr := comCall(b.toast.notifier, 8, uintptr(unsafe.Pointer(&setting))); failed(hr) {
		done(platform.NotificationAuthorizationUnknown, hresultError("ToastNotifier.Setting", hr))
		return
	}
	switch setting {
	case 0:
		done(platform.NotificationAuthorizationAllowed, nil)
	case 1, 2, 3, 4:
		done(platform.NotificationAuthorizationDenied, nil)
	default:
		done(platform.NotificationAuthorizationUnknown, errors.New("mygo: unknown toast notification setting"))
	}
}
func (b *Backend) RequestNotificationPermission(done func(platform.NotificationAuthorization, error)) {
	// Unpackaged WinRT toast has no runtime consent prompt. Never rewrite OS
	// notification preferences to simulate authorization.
	b.NotificationPermission(done)
}
func (b *Backend) ShowNotification(n *platform.Notification, done func(error)) {
	b.NotificationPermission(func(s platform.NotificationAuthorization, err error) {
		if err != nil {
			done(err)
			return
		}
		if s != platform.NotificationAuthorizationAllowed {
			done(platform.ErrNotificationsDenied)
			return
		}
		done(b.showToast(n))
	})
}
func (b *Backend) showToast(n *platform.Notification) error {
	payload, err := wire.ToastXML(n)
	if err != nil {
		return err
	}
	document, err := activateRuntime("Windows.Data.Xml.Dom.XmlDocument")
	if err != nil {
		return err
	}
	defer release(document)
	io := queryInterface(document, &iidXMLDocumentIO)
	if io == 0 {
		return errors.New("mygo: XML document loader unavailable")
	}
	defer release(io)
	xml, err := hstring(payload)
	if err != nil {
		return err
	}
	defer procWindowsDeleteString.Call(xml)
	if hr := comCall(io, 6, xml); failed(hr) {
		return hresultError("XmlDocument.LoadXml", hr)
	}
	factory, err := activationFactory("Windows.UI.Notifications.ToastNotification", &iidToastFactory)
	if err != nil {
		return err
	}
	defer release(factory)
	var toast uintptr
	if hr := comCall(factory, 6, document, uintptr(unsafe.Pointer(&toast))); failed(hr) {
		return hresultError("CreateToastNotification", hr)
	}
	defer release(toast)
	tagged := queryInterface(toast, &iidToast2)
	if tagged == 0 {
		return errors.New("mygo: native toast tag/group unavailable")
	}
	defer release(tagged)
	tag, err := hstring(wire.Tag(n.ID))
	if err != nil {
		return err
	}
	defer procWindowsDeleteString.Call(tag)
	group, err := hstring(toastGroup)
	if err != nil {
		return err
	}
	defer procWindowsDeleteString.Call(group)
	if hr := comCall(tagged, 6, tag); failed(hr) {
		return hresultError("ToastNotification.Tag", hr)
	}
	if hr := comCall(tagged, 8, group); failed(hr) {
		return hresultError("ToastNotification.Group", hr)
	}
	if hr := comCall(b.toast.notifier, 6, toast); failed(hr) {
		return hresultError("ToastNotifier.Show", hr)
	}
	return nil
}
func removeToast(history uintptr, appID, id string) error {
	app, err := hstring(appID)
	if err != nil {
		return err
	}
	defer procWindowsDeleteString.Call(app)
	if id == "" {
		if hr := comCall(history, 12, app); failed(hr) {
			return hresultError("ToastHistory.ClearWithId", hr)
		}
		return nil
	}
	tag, err := hstring(wire.Tag(id))
	if err != nil {
		return err
	}
	defer procWindowsDeleteString.Call(tag)
	group, err := hstring(toastGroup)
	if err != nil {
		return err
	}
	defer procWindowsDeleteString.Call(group)
	if hr := comCall(history, 8, tag, group, app); failed(hr) {
		return hresultError("ToastHistory.RemoveGroupedTagWithId", hr)
	}
	return nil
}
func (b *Backend) RemoveNotification(id string) {
	if b.toast != nil && b.toast.history != 0 {
		if err := removeToast(b.toast.history, b.toast.appID, id); err != nil {
			log.Print(err)
		}
	}
}
func (b *Backend) RemoveAllNotifications() {
	if b.toast != nil && b.toast.history != 0 {
		if err := removeToast(b.toast.history, b.toast.appID, ""); err != nil {
			log.Print(err)
		}
	}
}

// Only two callback tables are allocated for the lifetime of the process.
// COM-owned objects stay reachable until the last Release, including shutdown.
type toastCOMObject struct {
	vtbl    *uintptr
	state   *toastState
	b       *Backend
	factory bool
	target  *toastCOMObject
	refs    int32 // protected by toastObjectsMu
}

var (
	initToastCOMOnce    sync.Once
	toastFactoryVtbl    [5]uintptr
	toastActivationVtbl [4]uintptr
	toastObjectsMu      sync.Mutex
	toastObjects        = map[uintptr]*toastCOMObject{}
)

func newToastCOMObject(vtbl *uintptr, s *toastState, b *Backend, factory bool) *toastCOMObject {
	o := &toastCOMObject{vtbl: vtbl, state: s, b: b, factory: factory, refs: 1}
	toastObjectsMu.Lock()
	toastObjects[uintptr(unsafe.Pointer(o))] = o
	toastObjectsMu.Unlock()
	return o
}
func toastRef(this uintptr, delta int32) uintptr {
	toastObjectsMu.Lock()
	o := toastObjects[this]
	if o == nil {
		toastObjectsMu.Unlock()
		return 0
	}
	o.refs += delta
	refs := o.refs
	var target *toastCOMObject
	if refs == 0 {
		delete(toastObjects, this)
		target = o.target
	}
	toastObjectsMu.Unlock()
	if target != nil {
		release(uintptr(unsafe.Pointer(target)))
	}
	return uintptr(refs)
}
func boundedWide(p uintptr, max int) (string, bool) {
	if p == 0 {
		return "", false
	}
	units := make([]uint16, 0, max)
	for i := 0; i <= max; i++ {
		c := *(*uint16)(unsafe.Add(native(p), i*2))
		if c == 0 {
			return string(utf16.Decode(units)), true
		}
		units = append(units, c)
	}
	return "", false
}
func initToastCOM() {
	query := syscall.NewCallback(func(this, riid, out uintptr) uintptr {
		if out == 0 || riid == 0 {
			return 0x80004003
		}
		*(*uintptr)(native(out)) = 0
		o := (*toastCOMObject)(native(this))
		iid := *(*GUID)(native(riid))
		expected := iidNotificationActivation
		if o.factory {
			expected = iidClassFactory
		}
		if iid != iidUnknown && iid != expected {
			return eNoInterface
		}
		*(*uintptr)(native(out)) = this
		toastRef(this, 1)
		return sOK
	})
	add := syscall.NewCallback(func(this uintptr) uintptr { return toastRef(this, 1) })
	drop := syscall.NewCallback(func(this uintptr) uintptr { return toastRef(this, -1) })
	toastActivationVtbl = [4]uintptr{query, add, drop, syscall.NewCallback(func(this, app, args, input, count uintptr) uintptr {
		o := (*toastCOMObject)(native(this))
		appID, ok := boundedWide(app, 128)
		if !ok || appID != o.state.appID || count != 0 {
			return 0x80070057
		}
		arguments, ok := boundedWide(args, 2048)
		if !ok {
			return 0x80070057
		}
		id, action, err := wire.ParseActivation(arguments)
		if err != nil {
			return 0x80070057
		}
		if o.state.relay != nil {
			if err := o.state.relay(id, action); err != nil {
				log.Print(err)
				return 0x80004005
			}
			o.state.mu.Lock()
			o.state.closed = true
			o.state.mu.Unlock()
			return sOK
		}
		o.state.mu.Lock()
		if o.state.closed {
			o.state.mu.Unlock()
			return 0x80010108
		}
		if len(o.state.pending) >= 32 {
			o.state.mu.Unlock()
			return 0x8007000e
		}
		o.state.pending = append(o.state.pending, toastActivation{id, action})
		o.state.mu.Unlock()
		o.b.Signal()
		return sOK
	})}
	toastFactoryVtbl = [5]uintptr{query, add, drop, syscall.NewCallback(func(this, outer, riid, out uintptr) uintptr {
		if out == 0 {
			return 0x80004003
		}
		*(*uintptr)(native(out)) = 0
		if outer != 0 {
			return 0x80040110
		}
		o := (*toastCOMObject)(native(this))
		return comCall(uintptr(unsafe.Pointer(o.target)), 0, riid, out)
	}), syscall.NewCallback(func(this, lock uintptr) uintptr { return sOK })}
}
func (b *Backend) dispatchNotificationActions() {
	if b.toast == nil {
		return
	}
	b.toast.mu.Lock()
	pending := b.toast.pending
	b.toast.pending = nil
	b.toast.mu.Unlock()
	for _, a := range pending {
		b.h.NotificationAction(a.id, a.action)
	}
}
func validToastAppID(s string) bool {
	return s != "" && len(s) <= 128 && !strings.ContainsAny(s, "\\/\x00")
}
func toastRegistryKey(clsid string) string {
	return `Software\Classes\CLSID\{` + clsid + `}\LocalServer32`
}
func toastCommand(exe string) string { return `"` + exe + `" -Embedding` }

// RelayNotificationActivation services COM's launch while an existing process
// is still starting up. The callback forwards only validated IDs/actions; the
// helper never opens an application window or runs normal bootstrap.
func RelayNotificationActivation(forward func(id, action string) error) error {
	info, ok := (appController{}).Package()
	if !ok {
		return platform.ErrUnsupported
	}
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failed(hr) {
		return hresultError("CoInitializeEx notification relay", hr)
	}
	defer procCoUninitialize.Call()
	initToastCOMOnce.Do(initToastCOM)
	state := &toastState{appID: info.Identifier, relay: forward}
	state.activator = newToastCOMObject(&toastActivationVtbl[0], state, nil, false)
	state.factory = newToastCOMObject(&toastFactoryVtbl[0], state, nil, true)
	state.factory.target = state.activator
	addRef(uintptr(unsafe.Pointer(state.activator)))
	defer release(uintptr(unsafe.Pointer(state.factory)))
	defer release(uintptr(unsafe.Pointer(state.activator)))
	cls := guid(wire.ActivatorCLSID(info.Identifier))
	hr, _, _ = procCoRegisterClassObject.Call(uintptr(unsafe.Pointer(&cls)), uintptr(unsafe.Pointer(state.factory)), 4, 1, uintptr(unsafe.Pointer(&state.cookie)))
	if failed(hr) {
		return hresultError("register notification relay", hr)
	}
	defer procCoRevokeClassObject.Call(uintptr(state.cookie))
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		state.mu.Lock()
		finished := state.closed
		state.mu.Unlock()
		if finished {
			return nil
		}
		procMsgWaitForMultipleObjectsEx.Call(0, 0, 50, qsAllInput, mwmoInputAvailable)
		var m msg
		for {
			r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmRemove)
			if r == 0 {
				break
			}
			if m.Message == wmQuit {
				return errors.New("mygo: notification relay interrupted")
			}
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	return errors.New("mygo: notification COM activation relay timed out")
}
