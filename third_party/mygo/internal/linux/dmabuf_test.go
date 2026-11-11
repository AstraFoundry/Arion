//go:build linux && (amd64 || arm64)

package linux

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDisableDMABUF(t *testing.T) {
	LinkDMABUFProbe()
	defer func() { dmabufProbe = nil }()
	// A Wayland desktop on Intel's GPU: its socket, and a render node of
	// i915.
	runtime := t.TempDir()
	l, err := net.Listen("unix", filepath.Join(runtime, "wayland-1"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	os.WriteFile(filepath.Join(runtime, "wayland-stale"), nil, 0o644)
	host := func(t *testing.T, files map[string]string) string {
		root := t.TempDir()
		dev := filepath.Join(root, "sys", "class", "drm", "renderD128", "device")
		os.MkdirAll(dev, 0o755)
		if err := os.Symlink(filepath.Join("..", "..", "bus", "pci", "drivers", "i915"), filepath.Join(dev, "driver")); err != nil {
			t.Fatal(err)
		}
		for name, content := range files {
			p := filepath.Join(root, name)
			os.MkdirAll(filepath.Dir(p), 0o755)
			os.WriteFile(p, []byte(content), 0o644)
		}
		return root
	}
	wayland := map[string]string{"WAYLAND_DISPLAY": "wayland-1", "XDG_RUNTIME_DIR": runtime}
	with := func(env map[string]string, more ...string) map[string]string {
		e := map[string]string{}
		for k, v := range env {
			e[k] = v
		}
		for i := 0; i < len(more); i += 2 {
			e[more[i]] = more[i+1]
		}
		return e
	}
	for _, c := range []struct {
		name     string
		renderer string
		env      map[string]string
		files    map[string]string
		disable  bool
	}{
		{"default", "", wayland, nil, true},
		{"off", "off", wayland, nil, true},
		{"on", "on", nil, nil, false},
		{"the user's 1 with on", "on", with(nil, "WEBKIT_DISABLE_DMABUF_RENDERER", "1"), nil, false},
		{"the user's 0", "", with(nil, "WEBKIT_DISABLE_DMABUF_RENDERER", "0"), nil, false},
		{"the user's 0 with auto on NVIDIA", "auto", with(wayland, "WEBKIT_DISABLE_DMABUF_RENDERER", "0"), map[string]string{"sys/module/nvidia/version": ""}, false},
		{"auto on Wayland and Mesa", "auto", wayland, nil, false},
		{"auto with an absolute socket", "auto", map[string]string{"WAYLAND_DISPLAY": filepath.Join(runtime, "wayland-1")}, nil, false},
		{"auto with GDK_BACKEND=wayland,x11", "auto", with(wayland, "GDK_BACKEND", "wayland,x11"), nil, false},
		{"auto on X11", "auto", map[string]string{"DISPLAY": ":0"}, nil, true},
		{"auto with GDK_BACKEND=x11", "auto", with(wayland, "GDK_BACKEND", "x11"), nil, true},
		{"auto with a stale socket", "auto", with(wayland, "WAYLAND_DISPLAY", "wayland-stale"), nil, true},
		{"auto with a missing socket", "auto", with(wayland, "WAYLAND_DISPLAY", "wayland-9"), nil, true},
		{"auto without XDG_RUNTIME_DIR", "auto", with(wayland, "XDG_RUNTIME_DIR", ""), nil, true},
		{"auto with LIBGL_ALWAYS_SOFTWARE", "auto", with(wayland, "LIBGL_ALWAYS_SOFTWARE", "1"), nil, true},
		{"auto with LIBGL_ALWAYS_SOFTWARE=0", "auto", with(wayland, "LIBGL_ALWAYS_SOFTWARE", "0"), nil, false},
		{"auto with llvmpipe", "auto", with(wayland, "GALLIUM_DRIVER", "llvmpipe"), nil, true},
		{"auto on NVIDIA", "auto", wayland, map[string]string{"sys/module/nvidia/version": "550.0"}, true},
		{"auto on NVIDIA's proc", "auto", wayland, map[string]string{"proc/driver/nvidia/version": "NVRM"}, true},
		{"auto in Docker", "auto", wayland, map[string]string{".dockerenv": ""}, true},
		{"auto in Podman", "auto", wayland, map[string]string{"run/.containerenv": ""}, true},
		{"auto in WSL", "auto", wayland, map[string]string{"dev/dxg": ""}, true},
		{"auto in VirtualBox", "auto", wayland, map[string]string{"sys/class/dmi/id/product_name": "VirtualBox\n"}, true},
		{"auto in QEMU", "auto", wayland, map[string]string{"sys/class/dmi/id/product_name": "Standard PC (Q35 + ICH9, 2009)\n", "sys/class/dmi/id/sys_vendor": "QEMU\n"}, true},
		{"auto in Hyper-V", "auto", wayland, map[string]string{"sys/class/dmi/id/product_name": "Virtual Machine\n", "sys/class/dmi/id/sys_vendor": "Microsoft Corporation\n"}, true},
		{"auto on a laptop", "auto", wayland, map[string]string{"sys/class/dmi/id/product_name": "XPS 13 9380\n", "sys/class/dmi/id/sys_vendor": "Dell Inc.\n"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := host(t, c.files)
			getenv := func(k string) string { return c.env[k] }
			if got := disableDMABUF(c.renderer, getenv, root); got != c.disable {
				t.Errorf("disableDMABUF = %v, want %v", got, c.disable)
			}
		})
	}
	t.Run("auto without the probe", func(t *testing.T) {
		dmabufProbe = nil
		defer LinkDMABUFProbe()
		getenv := func(k string) string { return wayland[k] }
		if !disableDMABUF("auto", getenv, host(t, nil)) {
			t.Error("disableDMABUF = false, want true")
		}
	})
	t.Run("auto without a GPU", func(t *testing.T) {
		getenv := func(k string) string { return wayland[k] }
		if !disableDMABUF("auto", getenv, t.TempDir()) {
			t.Error("disableDMABUF = false, want true")
		}
	})
}
