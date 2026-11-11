package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// utf16Of encodes a string as a Windows resource holds it, to search a
// resource for it.
func utf16Of(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func TestLanguageTagForms(t *testing.T) {
	for _, tc := range []struct{ tag, desktop, lproj string }{
		{"en", "en", "en"},
		{"en-GB", "en_GB", "en-GB"},
		{"zh-Hans", "zh_CN", "zh-Hans"},
		{"zh-Hant", "zh_TW", "zh-Hant"},
		{"zh-Hant-HK", "zh_HK", "zh-HK"},
		{"zh-Hant-TW", "zh_TW", "zh-Hant"},
		{"zh-Hans-SG", "zh_SG", "zh-Hans"},
		{"zh", "zh_CN", "zh-Hans"},
		{"zh-TW", "zh_TW", "zh-Hant"},
		{"zh-HK", "zh_HK", "zh-HK"},
		{"zh-MO", "zh_MO", "zh-Hant"},
		{"pt-BR", "pt_BR", "pt-BR"},
		{"es-419", "es", "es-419"},
		{"sr-Latn", "sr@latin", "sr-Latn"},
		{"sr-Latn-RS", "sr_RS@latin", "sr-Latn"},
		{"sr-Cyrl-RS", "sr_RS", "sr-Cyrl"},
		{"uz-Cyrl", "uz@cyrillic", "uz-Cyrl"},
		{"de-ch", "de_CH", "de-CH"},
		{"iw", "he", "he"},
	} {
		if got := desktopTag(tc.tag); got != tc.desktop {
			t.Errorf("desktopTag(%q) = %q, want %q", tc.tag, got, tc.desktop)
		}
		if got := lprojName(tc.tag); got != tc.lproj {
			t.Errorf("lprojName(%q) = %q, want %q", tc.tag, got, tc.lproj)
		}
	}
}

func TestWindowsLanguageID(t *testing.T) {
	for _, tc := range []struct {
		tag string
		id  uint16
		ok  bool
	}{
		{"en", langEnUS, true},
		{"en-GB", 0x0809, true},
		{"zh-Hans", 0x0804, true},
		{"zh-Hans-CN", 0x0804, true},
		{"zh-Hant", 0x0404, true},
		{"zh-Hant-HK", 0x0C04, true},
		{"zh-TW", 0x0404, true},
		{"pt", 0x0416, true}, // Brazilian, as CLDR's pt
		{"pt-PT", 0x0816, true},
		{"sr-Latn-RS", 0x241A, true},
		{"tlh", 0, false},
	} {
		if id, ok := windowsLanguageID(tc.tag); id != tc.id || ok != tc.ok {
			t.Errorf("windowsLanguageID(%q) = %#04x, %v, want %#04x, %v", tc.tag, id, ok, tc.id, tc.ok)
		}
	}
}

func TestNSISLanguage(t *testing.T) {
	for _, tc := range []struct {
		tag, name string
		ok        bool
	}{
		{"en", "English", true},
		{"en-GB", "English", true},
		{"zh", "SimpChinese", true},
		{"zh-CN", "SimpChinese", true},
		{"zh-Hant", "TradChinese", true},
		{"zh-HK", "TradChinese", true},
		{"pt", "PortugueseBR", true},
		{"pt-PT", "Portuguese", true},
		{"sr-Latn-RS", "SerbianLatin", true},
		{"es-419", "SpanishInternational", true},
		{"nb-NO", "Norwegian", true},
		{"tlh", "", false},
		{"sw", "", false}, // Swahili, which NSIS 3 has no file of
	} {
		if name, ok := nsisLanguage(tc.tag); name != tc.name || ok != tc.ok {
			t.Errorf("nsisLanguage(%q) = %q, %v, want %q, %v", tc.tag, name, ok, tc.name, tc.ok)
		}
	}
	if got, err := nsisLanguageMacros(nil); err != nil || got != "!insertmacro MUI_LANGUAGE \"English\"\nLangString MYGO_DESKTOP_SHORTCUT ${LANG_ENGLISH} \"Create a desktop shortcut\"\n" {
		t.Errorf("an app without languages: %q, %v", got, err)
	}
	want := "!insertmacro MUI_LANGUAGE \"English\"\n!insertmacro MUI_LANGUAGE \"SimpChinese\"\n!insertmacro MUI_LANGUAGE \"TradChinese\"\n" +
		"LangString MYGO_DESKTOP_SHORTCUT ${LANG_ENGLISH} \"Create a desktop shortcut\"\n" +
		"LangString MYGO_DESKTOP_SHORTCUT ${LANG_SIMPCHINESE} \"创建桌面快捷方式\"\n" +
		"LangString MYGO_DESKTOP_SHORTCUT ${LANG_TRADCHINESE} \"建立桌面捷徑\"\n"
	if got, err := nsisLanguageMacros([]appLanguage{{Tag: "en"}, {Tag: "zh-Hans"}, {Tag: "zh-CN"}, {Tag: "zh-Hant"}}); err != nil || got != want {
		t.Errorf("the installer's languages are\n%swant\n%s(%v)", got, want, err)
	}
	if _, err := nsisLanguageMacros([]appLanguage{{Tag: "en"}, {Tag: "sw"}}); err == nil || !strings.Contains(err.Error(), "sw") || !strings.Contains(err.Error(), "zh-hans") {
		t.Errorf("a language NSIS has no file of: %v", err)
	}
}

// TestNSISDesktopShortcut checks the finish page's label from the app's
// catalogs, MyGo's translations and the English source, in that order.
func TestNSISDesktopShortcut(t *testing.T) {
	for _, tc := range []struct {
		name, source, english string
	}{
		{"translations", `{}`, `"Create a desktop shortcut"`},
		{"source override", `{"mygo.desktopShortcut": "Put it on my desktop"}`, `"Put it on my desktop"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := project(t, `{"name": "Notes", "i18n": {"source": "en"}}`, map[string]string{
				"en.json":         tc.source,
				"zh-Hans.json":    `{}`,
				"zh-Hant-HK.json": `{}`,
				"de.json":         `{"mygo.desktopShortcut": "Desktop-Verknüpfung \"Notizen\" für $USER erstellen"}`,
				"fr.json":         `{}`,
			})
			langs, err := c.languages()
			if err != nil {
				t.Fatal(err)
			}
			script, err := nsisLanguageMacros(langs)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				`LangString MYGO_DESKTOP_SHORTCUT ${LANG_ENGLISH} ` + tc.english,
				`LangString MYGO_DESKTOP_SHORTCUT ${LANG_SIMPCHINESE} "创建桌面快捷方式"`,
				`LangString MYGO_DESKTOP_SHORTCUT ${LANG_TRADCHINESE} "建立桌面捷徑"`,
				`LangString MYGO_DESKTOP_SHORTCUT ${LANG_GERMAN} "Desktop-Verknüpfung $\"Notizen$\" für $$USER erstellen"`,
				`LangString MYGO_DESKTOP_SHORTCUT ${LANG_FRENCH} "Create a desktop shortcut"`,
			} {
				if n := strings.Count(script, want+"\n"); n != 1 {
					t.Errorf("the installer's languages have %d of %q, want one", n, want)
				}
			}
			if strings.Count(script, "LangString MYGO_DESKTOP_SHORTCUT ") != 5 {
				t.Errorf("the installer's labels:\n%s", script)
			}
			// Regional catalogs sharing one NSIS language get one label,
			// from the first catalog, just as they get one MUI_LANGUAGE.
			script, err = nsisLanguageMacros(append(append(langs, langs...), appLanguage{Tag: "zh-Hans-CN"}))
			if err != nil || strings.Count(script, "LangString MYGO_DESKTOP_SHORTCUT ") != 5 || strings.Count(script, "!insertmacro MUI_LANGUAGE ") != 5 {
				t.Errorf("duplicate languages: %v\n%s", err, script)
			}
		})
	}
	script, err := nsisLanguageMacros(nil)
	if err != nil || !strings.Contains(script, `LangString MYGO_DESKTOP_SHORTCUT ${LANG_ENGLISH} "Create a desktop shortcut"`) {
		t.Errorf("an app without catalogs: %v\n%s", err, script)
	}
}

// TestDesktopShortcutCatalog checks that mygo generate takes
// mygo.desktopShortcut in an app's catalogs, as a key of MyGo's strings.
func TestDesktopShortcutCatalog(t *testing.T) {
	dir := testModule(t, map[string]string{
		"main.go":         "package main\nimport _ \"example.com/app/locales\"\nfunc main() {}\n",
		"mygo.json":       `{"i18n": {}}`,
		"locales/en.json": `{"mygo.desktopShortcut": "Put it on my desktop"}`,
		"locales/de.json": `{"mygo.desktopShortcut": "Desktop-Verknüpfung erstellen"}`,
	})
	if err := runGenerate([]string{dir}); err != nil {
		t.Fatalf("mygo generate must accept mygo.desktopShortcut: %v", err)
	}
}

// project writes a project whose catalogs are files, by name.
func project(t *testing.T, config string, catalogs map[string]string) *Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, jsonConfig), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range catalogs {
		if err := os.WriteFile(filepath.Join(dir, "locales", name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLanguagesFromCatalogs(t *testing.T) {
	c := project(t, `{"name": "Notes", "linux": {"comment": "Take notes"}, "i18n": {}}`, map[string]string{
		"fr.json":      `{"@@locale": "fr", "save": "Enregistrer"}`,
		"en.json":      `{"@@locale": "en", "save": "Save", "mygo.appName": "Ignored"}`,
		"zh-Hans.json": `{"mygo.appName": "笔记", "mygo.appDescription": "记笔记"}`,
		"notes.txt":    "not a catalog",
	})
	langs, err := c.languages()
	if err != nil {
		t.Fatal(err)
	}
	want := []appLanguage{{Tag: "en"}, {Tag: "fr"}, {Tag: "zh-Hans", Name: "笔记", Description: "记笔记"}}
	if len(langs) != len(want) {
		t.Fatalf("languages: %+v", langs)
	}
	for i := range want {
		if langs[i] != want[i] {
			t.Errorf("language %d is %+v, want %+v", i, langs[i], want[i])
		}
	}
	// Without the configuration, the app declares none, as before.
	if langs, err := project(t, `{"name": "Notes"}`, map[string]string{"fr.json": "{}"}).languages(); err != nil || langs != nil {
		t.Errorf("without i18n: %+v, %v", langs, err)
	}
	for what, files := range map[string]map[string]string{
		"not a language tag": {"french.json": "{}"},
		"unexpected end":     {"fr.json": "{"},
		"break the line":     {"fr.json": `{"mygo.appName": "Notes\nFR"}`},
	} {
		if _, err := project(t, `{"i18n": {"source": "en"}}`, files).languages(); err == nil || !strings.Contains(err.Error(), what) {
			t.Errorf("%v: %v", files, err)
		}
	}
}

// TestLanguagePackaging checks what a build tells each system of the
// languages of the app.
func TestLanguagePackaging(t *testing.T) {
	c := project(t, `{"name": "Notes", "identifier": "com.example.notes", "version": "1.2.3", "linux": {"comment": "Take notes"}, "i18n": {"dir": "locales", "source": "en"}}`, map[string]string{
		"en.json":         `{}`,
		"zh-Hans.json":    `{"mygo.appName": "笔记", "mygo.appDescription": "记笔记"}`,
		"zh-Hant-HK.json": `{"mygo.appName": "筆記"}`,
		"fr.json":         `{"mygo.appName": "Notes"}`,
		"sr-Latn.json":    `{"mygo.appDescription": "Beleške"}`,
	})

	// macOS: the development region, which the system falls back to, the
	// localizations, named as Xcode names them, and the app's name in each.
	plist := string(infoPlist(c, "Notes", "", ""))
	for _, want := range []string{"<key>CFBundleDevelopmentRegion</key>\n\t<string>en</string>", "<key>CFBundleLocalizations</key>", "<string>zh-Hans</string>", "<string>zh-HK</string>", "<string>sr-Latn</string>"} {
		if !strings.Contains(plist, want) {
			t.Errorf("the Info.plist has no %q:\n%s", want, plist)
		}
	}
	langs, _ := c.languages()
	res := t.TempDir()
	// The app's resources translate the reasons of Info.plist themselves, as Xcode's projects do.
	os.MkdirAll(filepath.Join(res, "zh-Hans.lproj"), 0o755)
	os.WriteFile(filepath.Join(res, "zh-Hans.lproj", "InfoPlist.strings"), []byte(`"NSMicrophoneUsageDescription" = "用麦克风录语音笔记。";`), 0o644)
	os.MkdirAll(filepath.Join(res, "fr.lproj"), 0o755)
	os.WriteFile(filepath.Join(res, "fr.lproj", "InfoPlist.strings"), []byte(`NSMicrophoneUsageDescription = "Enregistrer des notes vocales.";`), 0o644)
	// A link among the resources is copied as a link: the build must not write through it.
	linked := filepath.Join(t.TempDir(), "InfoPlist.strings")
	os.WriteFile(linked, []byte(`"NSCameraUsageDescription" = "Kamera";`), 0o644)
	os.MkdirAll(filepath.Join(res, "de.lproj"), 0o755)
	os.Symlink(linked, filepath.Join(res, "de.lproj", "InfoPlist.strings"))
	if err := writeLocalizations(c, res, append(langs, appLanguage{Tag: "de", Name: "Notizen"})); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(linked); string(data) != `"NSCameraUsageDescription" = "Kamera";` {
		t.Errorf("the build wrote through a link among the resources:\n%s", data)
	}
	if data, _ := os.ReadFile(filepath.Join(res, "de.lproj", "InfoPlist.strings")); !strings.Contains(string(data), "<string>Kamera</string>") || !strings.Contains(string(data), "<string>Notizen</string>") {
		t.Errorf("de.lproj/InfoPlist.strings:\n%s", data)
	}
	for lproj, want := range map[string]string{
		"zh-Hans": "<key>NSMicrophoneUsageDescription</key>\n\t<string>用麦克风录语音笔记。</string>",
		"fr":      "<key>NSMicrophoneUsageDescription</key>\n\t<string>Enregistrer des notes vocales.</string>",
	} {
		if data, _ := os.ReadFile(filepath.Join(res, lproj+".lproj", "InfoPlist.strings")); !strings.Contains(string(data), want) {
			t.Errorf("the app's InfoPlist.strings of %s is not kept:\n%s", lproj, data)
		}
	}
	for lproj, name := range map[string]string{"en": "", "zh-Hans": "笔记", "zh-HK": "筆記", "fr": "", "sr-Latn": ""} {
		data, err := os.ReadFile(filepath.Join(res, lproj+".lproj", "InfoPlist.strings"))
		if err != nil {
			t.Errorf("%s: %v", lproj, err)
			continue
		}
		if has := strings.Contains(string(data), "CFBundleDisplayName"); has != (name != "") || name != "" && !strings.Contains(string(data), "<string>"+name+"</string>") {
			t.Errorf("the InfoPlist.strings of %s is\n%s", lproj, data)
		}
	}
	// Apps without translations keep their Info.plist.
	if p := string(infoPlist(&Config{Name: "A"}, "A", "", "")); strings.Contains(p, "CFBundleLocalizations") {
		t.Errorf("an app without languages declares some:\n%s", p)
	}

	// Linux: the name and comment of the languages that have their own,
	// keyed by the POSIX locales launchers match.
	entry := linuxDesktopEntry(c, "notes", "notes")
	for _, want := range []string{"Name=Notes\n", "Name[zh_CN]=笔记\n", "Name[zh_HK]=筆記\n", "Comment=Take notes\n", "Comment[zh_CN]=记笔记\n", "Comment[sr@latin]=Beleške\n"} {
		if !strings.Contains(entry, want) {
			t.Errorf("the desktop entry has no %q:\n%s", want, entry)
		}
	}
	if strings.Contains(entry, "[fr]") || strings.Contains(entry, "[en]") {
		t.Errorf("the desktop entry names the app as it is named:\n%s", entry)
	}

	// Windows: a string table per language, the app's own first, which the
	// other resources are in.
	wl := windowsLanguages(c, langs)
	if len(wl) != 5 || wl[0] != (windowsLanguage{id: langEnUS, name: "Notes"}) || wl[3] != (windowsLanguage{id: 0x0804, name: "笔记"}) {
		t.Fatalf("the Windows languages are %+v", wl)
	}
	ver := versionInfo(c, wl)
	for _, want := range []string{"040904b0", "080404b0", "0c0404b0", "笔记", "筆記", "Translation"} {
		if !bytes.Contains(ver, utf16Of(want)) {
			t.Errorf("the version resource has no %q", want)
		}
	}
	if want := []byte{0x09, 0x04, 0xB0, 0x04, 0x0C, 0x04, 0xB0, 0x04}; !bytes.Contains(ver, want) {
		t.Errorf("the version resource does not list en-US then fr, Unicode: % x", ver)
	}
	syso, err := winresSyso(c, "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(syso, utf16Of("080404b0")) {
		t.Error("the Windows resources have no strings of zh-Hans")
	}
	// The installer is in each language NSIS has.
	if got, err := nsisLanguageMacros(langs); err != nil || !strings.HasPrefix(got, "!insertmacro MUI_LANGUAGE \"English\"\n") || !strings.Contains(got, "SerbianLatin") {
		t.Errorf("the installer's languages: %q, %v", got, err)
	}
}
