//go:build linux && (amd64 || arm64)

package e2e

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/egoist/mygo"
)

// On a desktop whose dark style is only in the settings portal, as
// GNOME's, pages are dark from the first, and follow the portal's changes
// while the app has not chosen an appearance (#1120). The app runs on a
// bus of its own, with a fake portal.
func TestPageColorSchemePortal(t *testing.T) {
	daemon, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is needed for a bus of the test's own")
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "bus.conf")
	// A bus without services to start, so that nothing but the fake answers
	// for the portal.
	if err := os.WriteFile(config, []byte(`<!DOCTYPE busconfig PUBLIC "-//freedesktop//DTD D-Bus Bus Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/dbus/1.0/busconfig.dtd">
<busconfig>
  <type>session</type>
  <listen>unix:path=`+filepath.Join(dir, "bus")+`</listen>
  <policy context="default">
    <allow send_destination="*" eavesdrop="true"/>
    <allow eavesdrop="true"/>
    <allow own="*"/>
  </policy>
</busconfig>
`), 0o644); err != nil {
		t.Fatal(err)
	}
	bus := exec.CommandContext(t.Context(), daemon, "--config-file="+config, "--nofork", "--print-address")
	busOut, err := bus.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := bus.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { bus.Process.Kill(); bus.Wait() }()
	// The daemon prints its address once it listens.
	address, err := bufio.NewReader(busOut).ReadString('\n')
	if err != nil {
		t.Fatalf("dbus-daemon: %v", err)
	}
	env := append(os.Environ(), "DBUS_SESSION_BUS_ADDRESS="+strings.TrimSpace(address))

	var portalErr bytes.Buffer
	portal := exec.CommandContext(t.Context(), os.Args[0])
	portal.Env = append(env, "MYGO_E2E_FAKE_PORTAL=1")
	portal.Stderr = &portalErr
	portalIn, err := portal.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	portalOut, err := portal.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := portal.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { portalIn.Close(); portal.Wait() }()
	// It says ready once it owns the portal's name.
	if line, err := bufio.NewReader(portalOut).ReadString('\n'); line != "ready\n" {
		portalIn.Close()
		portal.Wait()
		t.Fatalf("the fake portal: %q, %v\n%s", line, err, portalErr.String())
	}

	// The app asks for the portal's changes on its output, which the test
	// passes on. The helper's waits have their guard, after which it fails
	// and says what it waited for.
	app := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPageColorSchemePortalApp$", "-test.v")
	app.Env = append(env, "MYGO_E2E_PORTAL_APP=1")
	appOut, err := app.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	app.Stderr = app.Stdout
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	var log strings.Builder
	lines := bufio.NewScanner(appOut)
	for lines.Scan() {
		line := lines.Text()
		log.WriteString(line + "\n")
		if scheme, ok := strings.CutPrefix(line, "portal: "); ok {
			fmt.Fprintln(portalIn, scheme)
		}
	}
	if err := app.Wait(); err != nil {
		t.Errorf("%v:\n%s\nthe fake portal:\n%s", err, log.String(), portalErr.String())
	}
}

// TestPageColorSchemePortalApp is TestPageColorSchemePortal's app, which
// starts on a dark desktop.
func TestPageColorSchemePortalApp(t *testing.T) {
	if os.Getenv("MYGO_E2E_PORTAL_APP") == "" {
		t.Skip("TestPageColorSchemePortal runs it, on a bus of its own")
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Portal color scheme", Width: 300, Height: 200})
	w.Page().LoadHTML(`<meta name="color-scheme" content="light dark"><p>scheme</p>`, "")
	waitFor(t, w, "document.readyState === 'complete'")
	var updates atomic.Int32
	defer mygo.Theme.OnUpdated(func() { updates.Add(1) })()
	// setPortal has the fake portal change the color scheme (1 dark, 2
	// light, 0 no preference), and waits for the app to hear of it.
	setPortal := func(scheme int) {
		t.Helper()
		n := updates.Load()
		fmt.Printf("portal: %d\n", scheme)
		eventually(t, fmt.Sprintf("Theme.OnUpdated after the portal's color scheme %d", scheme), func() bool { return updates.Load() > n })
	}
	expect := func(what string, dark bool) {
		t.Helper()
		waitFor(t, w, fmt.Sprintf("matchMedia('(prefers-color-scheme: dark)').matches === %v", dark))
		if got := mygo.Theme.IsDark(); got != dark {
			t.Errorf("%s: Theme.IsDark() = %v", what, got)
		}
	}

	expect("a dark desktop", true)
	setPortal(2)
	expect("a light desktop", false)
	setPortal(1)
	expect("a dark desktop again", true)
	// Without a preference, GTK's theme decides: Adwaita is light.
	setPortal(0)
	expect("no preference", false)

	// The appearance the app chooses wins over the portal's.
	mygo.Theme.SetSource(mygo.ThemeLight)
	setPortal(1)
	expect("ThemeLight on a dark desktop", false)
	mygo.Theme.SetSource(mygo.ThemeSystem)
	expect("ThemeSystem on a dark desktop", true)
	mygo.Theme.SetSource(mygo.ThemeDark)
	setPortal(2)
	expect("ThemeDark on a light desktop", true)
	mygo.Theme.SetSource(mygo.ThemeSystem)
	expect("ThemeSystem on a light desktop", false)
}

// fakePortal serves the settings portal's color scheme on the session bus
// (org.freedesktop.portal.Settings), dark at first, and changes it to each
// number its input gives, with the SettingChanged signal.
func fakePortal() {
	runtime.LockOSThread()
	gio, err := purego.Dlopen("libgio-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var (
		busGetSync          func(busType int32, cancellable uintptr, gerr *uintptr) uintptr
		nodeInfoNewForXML   func(xml string, gerr *uintptr) uintptr
		nodeInfoLookup      func(node uintptr, name string) uintptr
		registerObject      func(conn uintptr, path string, iface uintptr, vtable unsafe.Pointer, data, free uintptr, gerr *uintptr) uint32
		callSync            func(conn uintptr, name, path, iface, method string, params, replyType uintptr, flags, timeout int32, cancellable uintptr, gerr *uintptr) uintptr
		emitSignal          func(conn uintptr, dest uintptr, path, iface, signal string, params uintptr, gerr *uintptr) bool
		variantParse        func(typ uintptr, text string, limit, endptr uintptr, gerr *uintptr) uintptr
		variantPrint        func(v uintptr, annotate bool) string
		returnValue         func(invocation, v uintptr)
		returnDBusError     func(invocation uintptr, name, message string)
		mainLoopNew         func(ctx uintptr, running bool) uintptr
		mainLoopRun         func(loop uintptr)
		scheme              atomic.Uint32
		vtable              [11]uintptr // GDBusInterfaceVTable: method_call, get_property, set_property, padding
		settingsIface       = "org.freedesktop.portal.Settings"
		portalPath          = "/org/freedesktop/portal/desktop"
		appearance, colorID = "org.freedesktop.appearance", "color-scheme"
	)
	for name, fn := range map[string]any{
		"g_bus_get_sync": &busGetSync, "g_dbus_node_info_new_for_xml": &nodeInfoNewForXML,
		"g_dbus_node_info_lookup_interface": &nodeInfoLookup, "g_dbus_connection_register_object": &registerObject,
		"g_dbus_connection_call_sync": &callSync, "g_dbus_connection_emit_signal": &emitSignal,
		"g_variant_parse": &variantParse, "g_variant_print": &variantPrint,
		"g_dbus_method_invocation_return_value": &returnValue, "g_dbus_method_invocation_return_dbus_error": &returnDBusError,
		"g_main_loop_new": &mainLoopNew, "g_main_loop_run": &mainLoopRun,
	} {
		purego.RegisterLibFunc(fn, gio, name)
	}
	cstr := func(p uintptr) string {
		base := *(*unsafe.Pointer)(unsafe.Pointer(&p))
		var b []byte
		for c := (*byte)(base); *c != 0; c = (*byte)(unsafe.Add(unsafe.Pointer(c), 1)) {
			b = append(b, *c)
		}
		return string(b)
	}
	fail := func(what string, gerr uintptr) {
		msg := "unknown error"
		if gerr != 0 {
			msg = cstr(*(*uintptr)(unsafe.Add(*(*unsafe.Pointer)(unsafe.Pointer(&gerr)), 8))) // GError's message
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", what, msg)
		os.Exit(1)
	}
	parse := func(text string) uintptr {
		var gerr uintptr
		v := variantParse(0, text, 0, 0, &gerr)
		if v == 0 {
			fail("g_variant_parse "+text, gerr)
		}
		return v
	}

	var gerr uintptr
	conn := busGetSync(2, 0, &gerr) // G_BUS_TYPE_SESSION
	if conn == 0 {
		fail("the session bus", gerr)
	}
	node := nodeInfoNewForXML(`<node><interface name="`+settingsIface+`">
  <method name="Read"><arg type="s" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="out"/></method>
  <method name="ReadAll"><arg type="as" direction="in"/><arg type="a{sa{sv}}" direction="out"/></method>
  <signal name="SettingChanged"><arg type="s"/><arg type="s"/><arg type="v"/></signal>
</interface></node>`, &gerr)
	if node == 0 {
		fail("the portal's interface", gerr)
	}
	scheme.Store(1)
	vtable[0] = purego.NewCallback(func(conn, sender, path, iface, method, params, invocation, data uintptr) {
		args := variantPrint(params, false)
		switch cstr(method) {
		case "Read":
			if args != fmt.Sprintf("('%s', '%s')", appearance, colorID) {
				returnDBusError(invocation, "org.freedesktop.portal.Error.NotFound", "Requested setting not found")
				return
			}
			// The portal wraps the value in a second variant.
			returnValue(invocation, parse(fmt.Sprintf("(<<uint32 %d>>,)", scheme.Load())))
		case "ReadAll":
			returnValue(invocation, parse(fmt.Sprintf("({'%s': {'%s': <uint32 %d>}},)", appearance, colorID, scheme.Load())))
		}
	})
	if registerObject(conn, portalPath, nodeInfoLookup(node, settingsIface), unsafe.Pointer(&vtable), 0, 0, &gerr) == 0 {
		fail("the portal's object", gerr)
	}
	// DBUS_NAME_FLAG_DO_NOT_QUEUE; the reply is 1 for the primary owner.
	res := callSync(conn, "org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "RequestName",
		parse("('org.freedesktop.portal.Desktop', uint32 4)"), 0, 0, -1, 0, &gerr)
	if res == 0 {
		fail("the portal's name", gerr)
	}
	if reply := variantPrint(res, false); reply != "(1,)" {
		fmt.Fprintf(os.Stderr, "the portal's name: %s\n", reply)
		os.Exit(1)
	}
	fmt.Println("ready")
	go func() {
		lines := bufio.NewScanner(os.Stdin)
		for lines.Scan() {
			var s uint32
			if _, err := fmt.Sscan(lines.Text(), &s); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			scheme.Store(s)
			var gerr uintptr
			if !emitSignal(conn, 0, portalPath, settingsIface, "SettingChanged",
				parse(fmt.Sprintf("('%s', '%s', <uint32 %d>)", appearance, colorID, s)), &gerr) {
				fail("SettingChanged", gerr)
			}
		}
		os.Exit(0) // the test is done with it
	}()
	mainLoopRun(mainLoopNew(0, false))
	runtime.KeepAlive(&vtable)
}
