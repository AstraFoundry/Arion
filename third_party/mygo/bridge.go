//go:build !js

package mygo

import (
	"context"
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/egoist/mygo/internal/bridge"
	"github.com/egoist/mygo/internal/platform"
)

// What windows say to the bridge in their pages, and what they hear from
// it. Builds for browsers, whose windows show no pages, leave it out with
// the JSON it takes (bridge_js.go).

// bridgeScript returns the script that installs the bridge in the
// window's pages.
func (w *Window) bridgeScript() string {
	return bridge.Script(bridge.Config{
		Platform: jsPlatform(),
		WindowID: w.top().id, // the window a web view is in
		Version:  Version,
		Secret:   w.secret,
	})
}

// decodeResult decodes the JSON result of eval into v.
func decodeResult(raw []byte, v any) error { return json.Unmarshal(raw, v) }

const (
	// The expression form: compile errors mean code is not an expression.
	evalExpr = "try{return JSON.stringify({ok:true,v:await(\n%s\n)})}catch(e){return JSON.stringify({ok:false,e:String(e&&e.message||e)})}"
	// The statement form runs code as a function body.
	evalBody = "try{return JSON.stringify({ok:true,v:await(async()=>{\n%s\n})()})}catch(e){return JSON.stringify({ok:false,e:String(e&&e.message||e)})}"
)

// eval returns the JSON result of code, which shares the memory of the
// page's answer.
func (w *Window) eval(ctx context.Context, code string) (jsontext.Value, error) {
	if w.content != nil {
		return nil, errNoPage
	}
	type result struct {
		raw string
		err error
	}
	// Both the cancellation and the page may answer; the first one counts
	// and the other must not block.
	ch := make(chan result, 2)
	stop := context.AfterFunc(ctx, func() { deliver(ch, result{err: ctx.Err()}) })
	defer stop()

	ran := false
	w.do(func(n platform.Window) {
		ran = true
		expr := strings.TrimRight(code, " \t\r\n;")
		n.CallAsyncFunction(fmt.Sprintf(evalExpr, expr), func(res string, err error) {
			if err == nil {
				deliver(ch, result{raw: res})
				return
			}
			if !strings.HasPrefix(err.Error(), "SyntaxError") {
				// It may have run (and then the page navigated or
				// crashed): running it again could run it twice.
				deliver(ch, result{err: err})
				return
			}
			// Not an expression: run it as a function body instead. The
			// first attempt failed to compile, so nothing ran twice.
			onMain(func() {
				if w.native == nil {
					deliver(ch, result{err: errDestroyed})
					return
				}
				w.native.CallAsyncFunction(fmt.Sprintf(evalBody, code), func(res string, err error) {
					deliver(ch, result{raw: res, err: err})
				})
			})
		})
	})
	if !ran {
		return nil, errDestroyed
	}
	r := await(ch)
	if r.err != nil {
		return nil, r.err
	}
	var out struct {
		OK bool     `json:"ok"`
		V  rawValue `json:"v"`
		E  string   `json:"e"`
	}
	if err := json.Unmarshal(stringBytes(r.raw), &out); err != nil {
		return nil, fmt.Errorf("mygo: unexpected eval result %q: %w", r.raw, err)
	}
	if !out.OK {
		return nil, &EvalError{Message: out.E}
	}
	return jsontext.Value(out.V), nil
}

// sendTitleBar tells the page of a window with a hidden title bar the room
// its controls take. The backend's script tells the first page at document
// start; this keeps the current page, and later ones, up to date. Main
// thread only.
func (w *Window) sendTitleBar() {
	if w.content != nil || !w.hiddenTitleBar || w.native == nil {
		return
	}
	if msg, err := encodeEvent(bridge.TitleBarEvent, bridge.NewTitleBar(w.native.TitleBar(), w.native.Zoom())); err == nil {
		w.enqueue(msg, true)
	}
}

// handleMessage routes a message posted by the page bridge. It runs on the
// main thread.
func (w *Window) handleMessage(msg string) {
	msg, ok := strings.CutPrefix(msg, w.secret)
	if !ok {
		return // not from the bridge
	}
	// Calls are decoded and executed off the main thread. The page context
	// is captured here so a navigation cannot slip in between.
	if len(msg) > 12 && msg[:12] == `{"t":"call",` {
		go handleCall(w, w.pageContext(), msg, w.trusted)
		return
	}
	var m struct {
		T string  `json:"t"`
		X float64 `json:"x"` // drop
		Y float64 `json:"y"`
		C int64   `json:"c"` // channels
		K string  `json:"k"`
		N int64   `json:"n"`
	}
	if err := json.Unmarshal(stringBytes(msg), &m); err != nil {
		return
	}
	switch m.T {
	case "dom-ready":
		w.outMu.Lock()
		w.domReady = true
		w.outbox = append(w.outbox, w.held...)
		w.held = nil
		w.outMu.Unlock()
		// The script at document start may tell a later page the room of
		// the controls before a change.
		w.sendTitleBar()
		w.flush()
		fire(&w.onDOMReady)
		w.readyToShow()
	case "drag":
		if w.native != nil {
			w.native.StartDrag()
		}
	case "dblclick":
		if w.native != nil {
			w.native.TitleBarDoubleClicked()
		}
	case "escape":
		w.escape()
	case "drop":
		if w.native != nil {
			w.filesDropped(w.native.DroppedFiles(), int(m.X), int(m.Y))
		}
	case "chan-ack":
		if c := w.channel(m.C, m.K); c != nil {
			c.ack(m.N)
		}
	case "chan-close":
		w.pageClosedChannel(m.C, m.K)
	}
}

// findScript returns the script with which FindInPage asks the bridge.
func findScript(text string, opts FindOptions) string {
	t, _ := jsonv1.Marshal(text)
	o, _ := jsonv1.Marshal(map[string]bool{"matchCase": opts.MatchCase, "backward": opts.Backward, "next": opts.FindNext})
	return fmt.Sprintf("window.__mygo.find(%s, %s)", t, o)
}
