//go:build darwin

package darwin

import (
	"testing"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/egoist/mygo/internal/platform"
)

type webViewScrollHandler struct {
	platform.WindowHandler
	events []platform.SurfaceEvent
}

func (h *webViewScrollHandler) SurfaceEvent(ev platform.SurfaceEvent) bool {
	h.events = append(h.events, ev)
	return true
}

func TestEmbeddedWebViewScrollReachesSurface(t *testing.T) {
	load()
	withPool(func() {
		h := &webViewScrollHandler{}
		view := msgInitRect(send(class("NSView"), "alloc"), sel("initWithFrame:"), NSRect{Size: NSSize{Width: 400, Height: 300}})
		s := &surface{w: &window{h: h}, view: view}
		w := &window{host: &window{h: h, surface: s}, opts: &platform.WindowOptions{ScrollToContainer: true}}
		web := id(objc.GetClass("MyGoWebView"))
		if web == 0 {
			classDef("MyGoWebView", "NSView", nil, webViewMethods())
		}
		webView := msgInitRect(send(class("MyGoWebView"), "alloc"), sel("initWithFrame:"), NSRect{Size: NSSize{Width: 400, Height: 300}})
		w.web = webView
		if theBackend == nil {
			theBackend = &Backend{byWebView: make(map[id]*window)}
		}
		theBackend.byWebView[webView] = w
		defer delete(theBackend.byWebView, webView)
		var create func(uintptr) uintptr
		purego.RegisterLibFunc(&create, libCG, "CGEventCreate")
		purego.RegisterLibFunc(&cgEventSetType, libCG, "CGEventSetType")
		cg := create(0)
		var setDouble func(uintptr, uint32, float64)
		purego.RegisterLibFunc(&setDouble, libCG, "CGEventSetDoubleValueField")
		cgEventSetType(cg, 22) // kCGEventScrollWheel

		setDouble(cg, 11, -2) // kCGScrollWheelEventDeltaAxis1
		setDouble(cg, 12, 1)  // kCGScrollWheelEventDeltaAxis2
		event := send(class("NSEvent"), "eventWithCGEvent:", cg)
		if event == 0 || sendInt(event, "type") != 22 {
			t.Fatal("could not construct a native scroll-wheel event")
		}
		send(webView, "scrollWheel:", uintptr(event))
		if len(h.events) != 1 {
			t.Fatalf("surface scroll events = %d, want 1", len(h.events))
		}
		got := h.events[0]
		if got.Kind != platform.PointerScroll || got.DX != -40 || got.DY != 80 || got.Precise {
			t.Fatalf("surface scroll = %+v, want line-scaled pointer deltas", got)
		}
		var createScroll func(uintptr, uint32, uint32, int32, int32) uintptr
		purego.RegisterLibFunc(&createScroll, libCG, "CGEventCreateScrollWheelEvent")
		cg = createScroll(0, 0, 2, 3, 0) // kCGScrollEventUnitPixel
		event = send(class("NSEvent"), "eventWithCGEvent:", cg)
		send(webView, "scrollWheel:", uintptr(event))
		if len(h.events) != 2 {
			t.Fatalf("surface trackpad events = %d, want 2", len(h.events))
		}
		got = h.events[1]
		if got.Kind != platform.PointerScroll || got.DX != 0 || got.DY != -3 || !got.Precise {
			t.Fatalf("surface trackpad scroll = %+v, want precise pointer deltas", got)
		}
		w.opts.ScrollToContainer = false
		send(webView, "scrollWheel:", uintptr(event))
		if len(h.events) != 2 {
			t.Fatal("a page that scrolls itself lost its wheel to the native surface")
		}
		send(webView, "removeFromSuperview")
		send(webView, "release")
		send(view, "release")
	})
}
