package app

import (
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	clipboardfeature "arion.local/desktop/ui/features/clipboard"
	"arion.local/desktop/ui/pages/menubar"
	"arion.local/desktop/ui/pages/settings"
	"arion.local/desktop/ui/shared/copy"
	"arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo/ui"
)

func TestClipboardSettingsDisclosureAndRecorder(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, actions, store := fixture()
			state.Preferences.Clipboard = runtime.DefaultClipboardPreferences()
			state.Preferences.UILanguage = locale
			page := settings.NewPage()
			page.Section = settings.Clipboard
			tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, SettingsWidth, SettingsHeight)
			settle(tt)
			messages := store.Copy()
			if hasText(tt, messages.Text(copy.ClipboardShortcut)) || hasText(tt, messages.Text(copy.ClipboardCount)) {
				t.Fatal("disabled history exposes options")
			}
			snapshot(t, tt, "settings-clipboard-off-"+string(locale))
			toggle := func() {
				t.Helper()
				label, ok := tt.Find(messages.Text(copy.ClipboardEnabled))
				if !ok {
					t.Fatal("enable setting missing")
				}
				tt.ClickAt(SettingsWidth-60, label.Y+label.H/2)
				rest(tt, 80*time.Millisecond)
				phase := "closing"
				if state.Preferences.Clipboard.Enabled {
					phase = "opening"
				}
				snapshot(t, tt, "settings-clipboard-"+phase+"-"+string(locale))
				rest(tt, 170*time.Millisecond)
			}
			toggle()
			if !state.Preferences.Clipboard.Enabled || !hasText(tt, messages.Text(copy.ClipboardShortcut)) {
				t.Fatal("enabled options missing")
			}
			if hasText(tt, messages.Text(copy.ClipboardClearAll)) {
				t.Fatal("history management remains in settings")
			}
			snapshot(t, tt, "settings-clipboard-on-"+string(locale))
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "settings-clipboard-on-dark-"+string(locale))
			tt.SetDark(false)
			settle(tt)
			limits := state.Preferences.Clipboard.Limits
			choose := func(label copy.Key, value string) {
				t.Helper()
				bounds, ok := tt.Find(messages.Text(label))
				if !ok {
					t.Fatal("retention setting missing")
				}
				tt.ClickAt(SettingsWidth-100, bounds.Y+bounds.H/2)
				if err := tt.Click(value); err != nil {
					t.Fatal(err)
				}
				settle(tt)
			}
			choose(copy.ClipboardCount, messages.ClipboardEntries(500))
			choose(copy.ClipboardDays, messages.ClipboardDays(7))
			saved := state.Preferences.Clipboard.Limits
			if saved.Count != 500 || saved.Days != 7 || saved.Bytes != limits.Bytes || saved.ItemBytes != limits.ItemBytes {
				t.Fatal("retention preset changed internal byte bounds")
			}

			if err := tt.Click(messages.Text(copy.ClipboardShortcutEdit)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			if state.Hold.Recording == "" {
				t.Fatal("native recorder not started")
			}
			snapshot(t, tt, "settings-clipboard-recording-"+string(locale))
			before := state.Preferences.Clipboard.Shortcut
			state.Hold.RecordedKey = state.Preferences.HoldKey
			state.Hold.RecordedSeq++
			settle(tt)
			if state.Preferences.Clipboard.Shortcut != before || state.Hold.Recording == "" {
				t.Fatal("conflicting key accepted")
			}
			state.Hold.RecordedKey = "ctrl+shift+k"
			state.Hold.RecordedSeq++
			settle(tt)
			if state.Preferences.Clipboard.Shortcut != "ctrl+shift+k" || state.Hold.Recording != "" {
				t.Fatal("recorded chord not saved")
			}
			if err := tt.Click(messages.Text(copy.ClipboardShortcutEdit)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			toggle()
			if state.Hold.Recording != "" || hasText(tt, messages.Text(copy.ClipboardCount)) {
				t.Fatal("disabling retained recording or options")
			}
			toggle()
			if err := tt.Click(messages.Text(copy.ClipboardShortcutEdit)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			page.Section = settings.General
			settle(tt)
			if state.Hold.Recording != "" {
				t.Fatal("navigation retained recording")
			}
			if !actions.called("clipboard-shortcut:ctrl+shift+k") {
				t.Fatal("shortcut action missing")
			}
		})
	}
}

func TestClipboardCompactSearchHoverPinAndPaste(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, rec, store := fixture()
			state.Preferences.UILanguage = locale
			state.Preferences.Clipboard.Enabled = true
			a, b := runtime.ClipboardText("Alpha 项目"), runtime.ClipboardText("Beta text")
			state.Clipboard.Items = []runtime.Clip{{ID: a.ID(), Payload: a, Copied: time.Now()}, {ID: b.ID(), Payload: b, Copied: time.Now()}}
			panel := &clipboardfeature.Panel{}
			tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, clipboardfeature.Width, clipboardfeature.Height(state.Clipboard.Items))
			settle(tt)
			m := store.Copy()
			for _, key := range []copy.Key{copy.ClipboardPin, copy.ClipboardDelete, copy.ClipboardClearAll, copy.ClipboardAll, copy.ClipboardManage} {
				if hasText(tt, m.Text(key)) {
					t.Fatalf("compact picker contains %s", key)
				}
			}
			snapshot(t, tt, "clipboard-compact-"+string(locale))
			r, ok := tt.Find(a.Text)
			if !ok {
				t.Fatal("missing first row")
			}
			tt.Move(r.X+4, r.Y+r.H/2)
			settle(tt)
			if !hasText(tt, m.Text(copy.ClipboardPin)) {
				t.Fatal("hover did not reveal pin")
			}
			snapshot(t, tt, "clipboard-compact-pin-"+string(locale))
			if err := tt.Click(m.Text(copy.ClipboardPin)); err != nil {
				t.Fatal(err)
			}
			if !rec.called("clipboard-pin:"+a.ID()) || rec.called("clipboard-paste:"+a.ID()) {
				t.Fatal("pin also pasted")
			}
			tt.Move(0, 0)
			settle(tt)
			if hasText(tt, m.Text(copy.ClipboardPin)) {
				t.Fatal("pin remained after hover")
			}
			tt.Key(0, ui.KeyDown)
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !rec.called("clipboard-paste:" + b.ID()) {
				t.Fatal("keyboard paste did not follow selection")
			}
			if err := tt.Click(m.Text(copy.ClipboardSearch)); err != nil {
				t.Fatal(err)
			}
			settle(tt)
			tt.Type("ALPHA")
			settle(tt)
			if !hasText(tt, a.Text) || hasText(tt, b.Text) {
				t.Fatal("search did not filter")
			}
			snapshot(t, tt, "clipboard-compact-search-"+string(locale))
			rec.calls = nil
			tt.Key(0, ui.KeyEnter)
			settle(tt)
			if !rec.called("clipboard-paste:" + a.ID()) {
				t.Fatal("search Enter did not paste its matching item")
			}
			rec.calls = nil
			if err := tt.Click(a.Text); err != nil {
				t.Fatal(err)
			}
			if !rec.called("clipboard-paste:" + a.ID()) {
				t.Fatal("single click did not paste")
			}
			tt.Key(0, ui.KeyEscape)
			settle(tt)
			if rec.called("clipboard-close") || hasText(tt, m.Text(copy.ClipboardCloseSearch)) {
				t.Fatal("Escape did not close search first")
			}
			tt.Key(0, ui.KeyEscape)
			settle(tt)
			if !rec.called("clipboard-close") {
				t.Fatal("Escape did not close picker")
			}
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "clipboard-compact-dark-"+string(locale))
		})
	}
}

func TestClipboardInlineImageAndFileRows(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, rec, store := fixture()
			state.Preferences.UILanguage = locale
			imagePayload := runtime.ClipPayload{Kind: runtime.ClipImage, PNG: []byte("\x89PNG\r\n\x1a\n")}
			filePayload := runtime.ClipPayload{Kind: runtime.ClipFiles, Files: []string{"/workspace/Plan.pdf"}}
			groupPayload := runtime.ClipPayload{Kind: runtime.ClipFiles, Files: []string{"/workspace/Research notes.pdf", "/workspace/Results.csv"}}
			plainPayload := runtime.ClipPayload{Kind: runtime.ClipFiles, Files: []string{"/workspace/README"}}
			bitmap := image.NewRGBA(image.Rect(0, 0, 80, 40))
			for y := 0; y < 40; y++ {
				for x := 0; x < 80; x++ {
					bitmap.Set(x, y, color.RGBA{R: 35, G: 150, B: 175, A: 255})
				}
			}
			state.Clipboard.Items = []runtime.Clip{{ID: imagePayload.ID(), Payload: imagePayload, Copied: time.Now()}, {ID: filePayload.ID(), Payload: filePayload, Copied: time.Now()}, {ID: groupPayload.ID(), Payload: groupPayload}, {ID: plainPayload.ID(), Payload: plainPayload}}
			state.Clipboard.Meta = map[string]runtime.ClipboardMeta{imagePayload.ID(): {Thumbnail: ui.NewBitmap(bitmap)}, filePayload.ID(): {Files: []runtime.ClipboardFile{{Name: "Plan.pdf", Path: filePayload.Files[0], Available: true}}}}
			state.Clipboard.Meta[groupPayload.ID()] = runtime.ClipboardMeta{Files: []runtime.ClipboardFile{{Path: groupPayload.Files[0], Available: true}, {Path: groupPayload.Files[1], Available: true}}}
			state.Clipboard.Meta[plainPayload.ID()] = runtime.ClipboardMeta{Files: []runtime.ClipboardFile{{Path: plainPayload.Files[0], Available: true}}}
			panel := &clipboardfeature.Panel{}
			tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, clipboardfeature.Width, clipboardfeature.Height(state.Clipboard.Items))
			settle(tt)
			if !hasText(tt, "Plan") || !hasText(tt, "PDF") || !hasText(tt, "README") || !hasText(tt, store.Copy().Count(2)) {
				t.Fatal("file name, extension badge or group count missing")
			}
			snapshot(t, tt, "clipboard-compact-resources-"+string(locale))
			tt.SetDark(true)
			settle(tt)
			snapshot(t, tt, "clipboard-compact-resources-dark-"+string(locale))
			tt.SetDark(false)
			settle(tt)
			imageRow, ok := tt.Find(store.Copy().Text(copy.ClipboardImages))
			if !ok {
				t.Fatal("image row missing")
			}
			tt.Move(imageRow.X+16, imageRow.Y+imageRow.H/2)
			rest(tt, 500*time.Millisecond)
			if !rec.called("clipboard-hover:" + imagePayload.ID()) {
				t.Fatal("thumbnail did not request preview")
			}
			tt.Move(0, 0)
			settle(tt)
			if err := tt.Click("Plan.pdf"); err != nil {
				t.Fatal(err)
			}
			if !rec.called("clipboard-paste:" + filePayload.ID()) {
				t.Fatal("ordinary file cannot paste")
			}
			state.Clipboard.Meta[filePayload.ID()] = runtime.ClipboardMeta{Files: []runtime.ClipboardFile{{Directory: true, Available: true}}}
			rec.calls = nil
			settle(tt)
			_ = tt.Click("Plan.pdf")
			if rec.called("clipboard-paste:" + filePayload.ID()) {
				t.Fatal("folder was deliverable")
			}
		})
	}
}

func TestClipboardMenuBarOmitsLastDictation(t *testing.T) {
	state, _, store := fixture()
	state.Dictation.Final = "Private last transcript"
	page := &menubar.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { page.Render(c, store) }, PanelWidth, PanelHeight)
	settle(tt)
	if hasText(tt, state.Dictation.Final) || hasText(tt, store.Copy().Text(copy.PanelLastDictation)) {
		t.Fatal("menu panel exposes last dictation")
	}
	snapshot(t, tt, "menubar-without-last-dictation")
}

func TestClipboardFourRowsAndHoverPreview(t *testing.T) {
	state, rec, store := fixture()
	texts := []string{"Short text", "A long clipboard entry with enough words to be hidden in the narrow history row", "Third entry", "Fourth entry", "Fifth entry", "Sixth entry"}
	for _, text := range texts {
		payload := runtime.ClipboardText(text)
		state.Clipboard.Items = append(state.Clipboard.Items, runtime.Clip{ID: payload.ID(), Payload: payload})
	}
	panel := &clipboardfeature.Panel{}
	height := clipboardfeature.Height(state.Clipboard.Items)
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, clipboardfeature.Width, height)
	settle(tt)
	if clipboardfeature.Width != 270 || height > 200 {
		t.Fatal("picker did not shrink")
	}
	for _, text := range texts[:4] {
		r, ok := tt.Find(text)
		if !ok || r.Y < 40 || r.Y+r.H > float32(height) {
			t.Fatalf("visible row missing: %q", text)
		}
	}
	if r, ok := tt.Find(texts[4]); ok && r.Y < float32(height-6) {
		t.Fatal("fifth row visible before scrolling")
	}
	snapshot(t, tt, "clipboard-four-rows")
	short, _ := tt.Find(texts[0])
	tt.Move(short.X+12, short.Y+short.H/2)
	rest(tt, 500*time.Millisecond)
	if rec.called("clipboard-hover:" + state.Clipboard.Items[0].ID) {
		t.Fatal("fully visible text opened preview")
	}
	long, _ := tt.Find(texts[1])
	rec.calls = nil
	tt.Move(long.X+12, long.Y+long.H/2)
	settle(tt)
	if rec.called("clipboard-hover:" + state.Clipboard.Items[1].ID) {
		t.Fatal("preview opened without dwell")
	}
	rest(tt, 500*time.Millisecond)
	if !rec.called("clipboard-hover:" + state.Clipboard.Items[1].ID) {
		t.Fatal("truncated text did not open preview")
	}
	rec.calls = nil
	tt.Move(0, 0)
	settle(tt)
	if !rec.called("clipboard-hover:") {
		t.Fatal("preview did not close after leaving content")
	}
	for range 5 {
		tt.Key(0, ui.KeyDown)
	}
	settle(tt)
	last, ok := tt.Find(texts[5])
	if !ok || last.Y < 40 || last.Y+last.H > float32(height) {
		t.Fatal("keyboard selection did not scroll last row into view")
	}
	tt.Key(0, ui.KeyEnter)
	if !rec.called("clipboard-paste:" + state.Clipboard.Items[5].ID) {
		t.Fatal("offscreen history cannot paste after navigation")
	}
	snapshot(t, tt, "clipboard-four-rows-scrolled")
}

func TestClipboardHoverPreviewContent(t *testing.T) {
	state, rec, store := fixture()
	payload := runtime.ClipPayload{Kind: runtime.ClipFiles, Files: []string{"/workspace/Research notes/Quarterly design review.pdf"}}
	item := runtime.Clip{ID: payload.ID(), Payload: payload}
	state.Clipboard.Items = []runtime.Clip{item}
	panel := &clipboardfeature.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, clipboardfeature.Width, clipboardfeature.Height(state.Clipboard.Items))
	settle(tt)
	row, ok := tt.Find("Quarterly design review.pdf")
	if !ok {
		t.Fatal("file row missing")
	}
	tt.Move(row.X+12, row.Y+row.H/2)
	rest(tt, 500*time.Millisecond)
	if !rec.called("clipboard-hover:" + item.ID) {
		t.Fatal("file name did not request preview")
	}
	preview := ui.NewTester(func(c *ui.Context) { clipboardfeature.Preview(c, item, nil) }, clipboardfeature.PreviewWidth, clipboardfeature.PreviewHeight)
	settle(preview)
	if !hasText(preview, payload.Files[0]) {
		t.Fatal("preview omitted full file path")
	}
	snapshot(t, preview, "clipboard-hover-file-preview")
	text := runtime.ClipboardText("First line\nSecond line, preserving original whitespace")
	item = runtime.Clip{ID: text.ID(), Payload: text}
	preview = ui.NewTester(func(c *ui.Context) { clipboardfeature.Preview(c, item, nil) }, clipboardfeature.PreviewWidth, clipboardfeature.PreviewHeight)
	settle(preview)
	if !hasText(preview, text.Text) {
		t.Fatal("preview changed original text")
	}
	snapshot(t, preview, "clipboard-hover-text-preview")
	pixels := image.NewRGBA(image.Rect(0, 0, 640, 320))
	for y := range 320 {
		for x := range 640 {
			pixels.Set(x, y, color.RGBA{R: 35, G: 150, B: 175, A: 255})
		}
	}
	bitmap := ui.NewBitmap(pixels)
	item.Payload = runtime.ClipPayload{Kind: runtime.ClipImage}
	preview = ui.NewTester(func(c *ui.Context) { clipboardfeature.Preview(c, item, bitmap) }, clipboardfeature.PreviewWidth, clipboardfeature.PreviewHeight)
	settle(preview)
	snapshot(t, preview, "clipboard-hover-image-preview")
}

func TestMenuPanelHeightFollowsVisibleContent(t *testing.T) {
	for _, locale := range []runtime.UILanguage{runtime.LanguageEnglish, runtime.LanguageChinese} {
		t.Run(string(locale), func(t *testing.T) {
			state, _, store := fixture()
			state.Preferences.UILanguage = locale
			panel := &menubar.Panel{}
			tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, PanelWidth, PanelHeightFor(store))
			for _, variant := range []struct {
				name                        string
				scene, provisioning, update bool
			}{
				{name: "ready"}, {name: "scene", scene: true}, {name: "provisioning", provisioning: true}, {name: "update", update: true}, {name: "all", scene: true, provisioning: true, update: true},
			} {
				state.Preferences.Polish, state.Preferences.MatchApp = variant.scene, variant.scene
				state.Polish = runtime.Polish{State: runtime.PolishReady, Installed: true}
				state.SceneTarget = runtime.SceneTarget{Known: variant.scene, AppName: "Cursor", Hints: runtime.SceneHints{App: "com.todesktop.230313mzl4w4u92", Field: runtime.FieldText}}
				state.Model.State = runtime.ModelReady
				if variant.provisioning {
					state.Model.State, state.Model.Phase = runtime.ModelProvisioning, runtime.PhaseDownloading
				}
				state.Update = runtime.Update{}
				if variant.update {
					state.Update = runtime.Update{Phase: runtime.UpdateAvailable, Version: "0.2.0"}
				}
				height := PanelHeightFor(store)
				tt.SetSize(PanelWidth, height)
				rest(tt, 200*time.Millisecond)
				quit, ok := tt.Find(store.Copy().Text(copy.Quit))
				if !ok || quit.Y < 0 || quit.Y+quit.H > float32(height) {
					t.Fatalf("%s: footer clipped at height %d: %+v", variant.name, height, quit)
				}
				if variant.name == "ready" && height != 160 {
					t.Fatal("ready panel reserves empty transcript space")
				}
				snapshot(t, tt, "panel-compact-"+variant.name+"-"+string(locale))
			}
		})
	}
}

func TestClipboardUnlocatedTargetCopiesInsteadOfPasting(t *testing.T) {
	state, rec, store := fixture()
	p := runtime.ClipboardText("Copy-only fallback")
	state.Clipboard.CopyOnly = true
	state.Clipboard.Items = []runtime.Clip{{ID: p.ID(), Payload: p}}
	panel := &clipboardfeature.Panel{}
	tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, clipboardfeature.Width, clipboardfeature.Height(state.Clipboard.Items))
	settle(tt)
	if err := tt.Click(p.Text); err != nil {
		t.Fatal(err)
	}
	if !rec.called("clipboard-copy:"+p.ID()) || rec.called("clipboard-paste:"+p.ID()) {
		t.Fatal("unverified target caused paste")
	}
}

func TestClipboardActivationFollowsTarget(t *testing.T) {
	for _, copyOnly := range []bool{false, true} {
		for _, trigger := range []string{"enter", "search-enter", "return-icon", "row"} {
			t.Run(fmt.Sprintf("copy-only-%t/%s", copyOnly, trigger), func(t *testing.T) {
				state, rec, store := fixture()
				p := runtime.ClipboardText("Target-aware action")
				state.Clipboard.CopyOnly = copyOnly
				state.Clipboard.Items = []runtime.Clip{{ID: p.ID(), Payload: p}}
				panel := &clipboardfeature.Panel{}
				tt := ui.NewTester(func(c *ui.Context) { panel.Render(c, store) }, clipboardfeature.Width, clipboardfeature.Height(state.Clipboard.Items))
				settle(tt)
				action, other, label := "clipboard-paste:", "clipboard-copy:", copy.ClipboardPaste
				if copyOnly {
					action, other, label = other, action, copy.Copy
				}
				switch trigger {
				case "enter":
					tt.Key(0, ui.KeyEnter)
				case "search-enter":
					if err := tt.Click(store.Copy().Text(copy.ClipboardSearch)); err != nil {
						t.Fatal(err)
					}
					settle(tt)
					tt.Type("Target")
					settle(tt)
					tt.Key(0, ui.KeyEnter)
				case "return-icon":
					if err := tt.Click(store.Copy().Text(label)); err != nil {
						t.Fatal(err)
					}
				case "row":
					if err := tt.Click(p.Text); err != nil {
						t.Fatal(err)
					}
				}
				settle(tt)
				count := 0
				for _, call := range rec.calls {
					if call == action+p.ID() {
						count++
					}
				}
				if count != 1 || rec.called(other+p.ID()) {
					t.Fatalf("activation must use the target mode exactly once: %v", rec.calls)
				}
			})
		}
	}
}
