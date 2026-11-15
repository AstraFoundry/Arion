package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestNSISUninstallEntry checks the entry of the app in Settings > Apps.
func TestNSISUninstallEntry(t *testing.T) {
	key := `Software\Microsoft\Windows\CurrentVersion\Uninstall\com.example.app`
	c := &Config{Name: "My App", Version: "1.2.0", Copyright: "Copyright © 2024-2026 Example Inc. All rights reserved."}
	entry := nsisUninstallEntry(c, key, "My App.exe")
	for _, line := range []string{
		`  WriteRegStr HKCU "` + key + `" "DisplayName" "My App"` + "\n",
		`  WriteRegStr HKCU "` + key + `" "DisplayVersion" "1.2.0"` + "\n",
		`  WriteRegStr HKCU "` + key + `" "DisplayIcon" "$INSTDIR\My App.exe"` + "\n",
		`  WriteRegStr HKCU "` + key + `" "Publisher" "Example Inc."` + "\n",
		`  WriteRegStr HKCU "` + key + `" "UninstallString" '"$INSTDIR\Uninstall.exe"'` + "\n",
		`  WriteRegStr HKCU "` + key + `" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'` + "\n",
	} {
		if !strings.Contains(entry, line) {
			t.Errorf("the entry has no %q:\n%s", line, entry)
		}
	}
	c.Publisher = "Jane $1"
	if entry := nsisUninstallEntry(c, key, "My App.exe"); !strings.Contains(entry, `"Publisher" "Jane $$1"`) {
		t.Errorf("the entry does not name the configured publisher:\n%s", entry)
	}
	c.Publisher, c.Copyright = "", ""
	if entry := nsisUninstallEntry(c, key, "My App.exe"); strings.Contains(entry, "Publisher") {
		t.Errorf("the entry names a publisher without one:\n%s", entry)
	}
}

func TestConfigPublisher(t *testing.T) {
	for copyright, want := range map[string]string{
		"":                          "",
		"Copyright © 2026 HeroSizy": "HeroSizy",
		"© 2026 Example Inc.":       "Example Inc.",
		"(c) 2020-2026 Jane Doe":    "Jane Doe",
		"Copyright 2019 – present Acme, Inc. All rights reserved.": "Acme, Inc.",
		"Example Inc.": "Example Inc.",
	} {
		if got := (&Config{Copyright: copyright}).publisher(); got != want {
			t.Errorf("publisher of %q = %q, want %q", copyright, got, want)
		}
	}
	if got := (&Config{Copyright: "© 2026 Example Inc.", Publisher: " Jane "}).publisher(); got != "Jane" {
		t.Errorf("publisher = %q, want the configured one", got)
	}
}

// TestNSISCloseApp checks the functions with which the installer and the
// uninstaller quit the app, silent ones too.
func TestNSISCloseApp(t *testing.T) {
	script := nsisCloseApp(&Config{Name: "My App"})
	for _, s := range []string{"Function CloseApp\n", "Function un.CloseApp\n", "IfSilent close\n", "$$env:MYGO_APP_SCRIPT"} {
		if !strings.Contains(script, s) {
			t.Errorf("no %q in:\n%s", s, script)
		}
	}
	for _, s := range []string{
		"$_.ExecutablePath.StartsWith($dir, [StringComparison]::OrdinalIgnoreCase)",
		"$running | Stop-Process -Force",
	} {
		if !strings.Contains(closeAppPS1, s) {
			t.Errorf("closeAppPS1 has no %q", s)
		}
	}
	for i, r := range script + closeAppPS1 {
		if r >= 0x80 {
			t.Fatalf("%q at %d: not ASCII", r, i)
		}
	}
	// The installer writes the script, its lines escaped for NSIS, each
	// shorter than an NSIS string.
	var written strings.Builder
	first, _, _ := strings.Cut(script, "FunctionEnd")
	for _, line := range strings.Split(first, "\n") {
		if text, ok := strings.CutPrefix(line, "  FileWrite $2 "); ok {
			if len(text) > 1000 {
				t.Errorf("line of %d characters: %s", len(text), text)
			}
			text = strings.TrimSuffix(strings.TrimPrefix(text, `"`), `"`)
			written.WriteString(strings.NewReplacer(`$\r$\n`, "\n", `$\t`, "\t", `$\"`, `"`, `$$`, `$`).Replace(text))
		}
	}
	if got := written.String(); got != closeAppPS1 {
		t.Errorf("the installer writes\n%s", got)
	}
	if pwsh, err := exec.LookPath("pwsh"); err == nil {
		path := filepath.Join(t.TempDir(), "close.ps1")
		os.WriteFile(path, []byte(closeAppPS1), 0o644)
		if out, err := exec.Command(pwsh, "-NoProfile", "-Command", "$null = [scriptblock]::Create((Get-Content -Raw '"+path+"'))").CombinedOutput(); err != nil {
			t.Errorf("pwsh cannot parse closeAppPS1: %v\n%s", err, out)
		}
	}
}

// TestWindowsInstaller builds the installer of an app, on Windows with the
// NSIS that mygo build downloads when it is not installed, and elsewhere
// where NSIS is installed. Its sign command lists the files it signs: the
// executable, the uninstaller and the installer. On Windows, it installs
// and uninstalls the app silently: the install makes the Start menu
// shortcut but not the desktop one the finish page offers, and the
// uninstall removes both.
func TestWindowsInstaller(t *testing.T) {
	dir, work, artifacts := buildTestInstaller(t, map[string]string{
		"main.go":              "package main\nimport (\"os\"; \"time\")\nfunc main() { if len(os.Args)>1 { _ = os.WriteFile(os.Args[1], []byte(\"ready\"), 0600); for {time.Sleep(time.Second)} } }\n",
		"mygo.json":            `{"name": "Setup Test", "identifier": "com.example.setuptest", "version": "2.0.0", "urlSchemes": ["setuptest"], "fileAssociations": [{"ext": ["setuptest"], "name": "Setup Test File"}], "windows": {"signCommand": "echo %1>>signed.txt"}, "i18n": {}}`,
		"locales/en.json":      `{}`,
		"locales/zh-Hans.json": `{}`,
		"resources/data/a.txt": "a",
		"resources/b.txt":      "b",
		"resources/icon.png":   string(defaultIcon()),
	})
	setup := artifacts[len(artifacts)-1]
	if filepath.Base(setup) != "setup-test-setup-2.0.0-amd64.exe" {
		t.Fatalf("artifacts = %q", artifacts)
	}
	// The executable, the uninstaller that makensis made, then the
	// installer.
	listed, _ := os.ReadFile(filepath.Join(dir, "signed.txt"))
	var signed []string
	for _, line := range strings.Split(strings.TrimSpace(string(listed)), "\n") {
		signed = append(signed, filepath.Base(strings.Trim(line, "\r \"")))
	}
	if len(signed) != 3 || signed[0] != "Setup Test.exe" || signed[1] == signed[0] || signed[1] == signed[2] || signed[2] != "setup-test-setup-2.0.0-amd64.exe" {
		t.Errorf("signed %q", signed)
	}
	// The finish page uses a LangString, defined for every wizard language.
	nsi := filepath.Join(work, "installer.nsi")
	script, err := os.ReadFile(nsi)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`!define MUI_FINISHPAGE_SHOWREADME_TEXT "$(MYGO_DESKTOP_SHORTCUT)"`,
		`LangString MYGO_DESKTOP_SHORTCUT ${LANG_ENGLISH} "Create a desktop shortcut"`,
		`LangString MYGO_DESKTOP_SHORTCUT ${LANG_SIMPCHINESE} "创建桌面快捷方式"`,
	} {
		if n := strings.Count(string(script), want); n != 1 {
			t.Errorf("installer.nsi has %d of %q, want one", n, want)
		}
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return
	}
	// NSIS takes /D= unquoted, and Go quotes arguments with spaces.
	install := filepath.Join(t.TempDir(), "SetupTest")
	if out, err := exec.Command(setup, "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("installing: %v\n%s", err, out)
	}
	for _, f := range []string{"Setup Test.exe", "data/a.txt", "Uninstall.exe"} {
		if _, err := os.Stat(filepath.Join(install, filepath.FromSlash(f))); err != nil {
			t.Errorf("not installed: %s", f)
		}
	}
	// A silent install, as an update runs, quits the running app without
	// asking and then installs.
	ready := filepath.Join(t.TempDir(), "ready")
	running := exec.Command(filepath.Join(install, "Setup Test.exe"), ready)
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = running.Process.Kill() })
	deadline := time.Now().Add(10 * time.Second)
	for !fileExists(ready) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if !fileExists(ready) {
		t.Fatal("owned app did not start")
	}
	exited := make(chan struct{})
	go func() { _ = running.Wait(); close(exited) }()
	if out, err := exec.Command(setup, "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("installing over the running app: %v\n%s", err, out)
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("the silent install left the app running")
	}
	unknown := filepath.Join(install, "user-added.txt")
	if err := os.WriteFile(unknown, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	// A locked later payload entry must roll back entries already promoted.
	// This reproduces the old installer's mixed-version failure without
	// touching any real installation or user data.
	oldExecutable := filepath.Join(install, "Setup Test.exe")
	oldBytes, err := os.ReadFile(oldExecutable)
	if err != nil {
		t.Fatal(err)
	}
	previous := append(append([]byte{}, oldBytes...), []byte("previous release")...)
	if err := os.WriteFile(oldExecutable, previous, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, "b.txt"), []byte("previous b"), 0600); err != nil {
		t.Fatal(err)
	}
	locked, err := os.Open(filepath.Join(install, "data", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	_, installErr := exec.Command(setup, "/S", "/D="+install).CombinedOutput()
	_ = locked.Close()
	if installErr == nil {
		t.Fatal("installation succeeded while an installed resource was locked")
	}
	if got, err := os.ReadFile(oldExecutable); err != nil || !bytes.Equal(got, previous) {
		t.Fatalf("failed upgrade did not restore the previous executable: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(install, "b.txt")); err != nil || string(got) != "previous b" {
		t.Fatalf("failed upgrade did not restore the previous resource: %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(install, "data", "a.txt")); err != nil || string(got) != "a" {
		t.Fatalf("failed upgrade lost the previous resource: %q, %v", got, err)
	}
	if out, err := exec.Command(setup, "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("retrying unlocked upgrade: %v\n%s", err, out)
	}
	if got, err := os.ReadFile(oldExecutable); err != nil || !bytes.Equal(got, oldBytes) {
		t.Fatalf("successful upgrade did not replace the executable: %v", err)
	}
	if got, err := os.ReadFile(unknown); err != nil || string(got) != "preserve" {
		t.Fatalf("upgrade changed an unrelated file: %q, %v", got, err)
	}
	registered := func(key string) bool {
		return exec.Command("reg", "query", `HKCU\Software\Classes\`+key).Run() == nil
	}
	for _, key := range []string{`.setuptest\OpenWithProgids`, `com.example.setuptest.setuptest\shell\open\command`, `setuptest\shell\open\command`} {
		if !registered(key) {
			t.Errorf("the installer did not register %s", key)
		}
	}
	startMenu, desktop := filepath.Join(shellFolder(t, "Programs"), "Setup Test.lnk"), filepath.Join(shellFolder(t, "Desktop"), "Setup Test.lnk")
	if !fileExists(startMenu) {
		t.Errorf("the installer made no %s", startMenu)
	}
	if fileExists(desktop) {
		t.Errorf("the silent install made %s", desktop)
	}
	// Stands for the shortcut that the finish page makes.
	if err := os.WriteFile(desktop, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(desktop) })
	defer func() {
		for _, key := range []string{`com.example.setuptest.setuptest`, `setuptest`} {
			if registered(key) {
				t.Errorf("the uninstaller left %s", key)
			}
		}
		for _, f := range []string{startMenu, desktop} {
			if fileExists(f) {
				t.Errorf("the uninstaller left %s", f)
			}
		}
	}()
	uninstall(t, install)
	if got, err := os.ReadFile(unknown); err != nil || string(got) != "preserve" {
		t.Fatalf("uninstall changed an unrelated file: %q, %v", got, err)
	}
}

// TestSkipInstaller builds a Windows app with updates and -skip-installer,
// with a makensis first on PATH that leaves a mark and fails: the build
// never runs NSIS, makes no installer nor install.ps1, and has the update
// files to upload. Without the flag, the same build runs it.
func TestSkipInstaller(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles programs")
	}
	bin := t.TempDir()
	mark := filepath.Join(bin, "ran")
	fake, script := "makensis", "#!/bin/sh\ntouch '"+mark+"'\nexit 1\n"
	if runtime.GOOS == "windows" {
		fake, script = "makensis.bat", "@echo off\r\necho.>\""+mark+"\"\r\nexit /b 1\r\n"
	}
	if err := os.WriteFile(filepath.Join(bin, fake), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	pub, priv, _ := ed25519.GenerateKey(nil)
	t.Setenv("MYGO_UPDATER_PRIVATE_KEY", base64.StdEncoding.EncodeToString(priv))

	dir := testModule(t, map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"mygo.json": `{"name": "Skip Test", "version": "1.0.0", "updates": {"github": "me/skip-test", "deltas": 0, "publicKey": "` + base64.StdEncoding.EncodeToString(pub) + `"}}`,
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildOptions{sign: "-", skipInstaller: true, work: t.TempDir()}
	if opts.pkg, err = packageDir(c); err != nil {
		t.Fatal(err)
	}
	artifacts, err := buildPlatform(c, "windows", "amd64", opts)
	if err != nil {
		t.Fatal(err)
	}
	if fileExists(mark) {
		t.Error("the build with -skip-installer ran makensis")
	}
	names, err := entryNames(filepath.Join(dir, "dist", "windows-amd64"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		if strings.Contains(name, installerMark) || name == installPS1Name {
			t.Errorf("the build with -skip-installer made %s", name)
		}
	}
	files, manifests, err := uploads(artifacts)
	if err != nil || len(files) != 1 || filepath.Base(files[0]) != "skip-test-1.0.0-windows-amd64.tar.gz" || len(manifests) != 1 {
		t.Errorf("uploads of %q = %q, %q, %v", artifacts, files, manifests, err)
	}

	opts.skipInstaller = false
	if _, err := buildPlatform(c, "windows", "amd64", opts); err == nil || !strings.Contains(err.Error(), "makensis") || !fileExists(mark) {
		t.Errorf("the build without -skip-installer: %v, ran makensis: %v", err, fileExists(mark))
	}
}

// buildTestInstaller builds the Windows installer of a module of files
// for amd64, and returns the module's directory, the work directory the
// installer script is written to, and the artifacts, the installer last.
// It skips the test without NSIS.
func buildTestInstaller(t *testing.T, files map[string]string) (string, string, []string) {
	t.Helper()
	if testing.Short() {
		t.Skip("compiles programs")
	}
	if tool, err := nsisCompiler(); err != nil {
		t.Fatal(err)
	} else if tool == "" {
		t.Skip("NSIS is not installed")
	}
	dir := testModule(t, files)
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildOptions{sign: "-", work: t.TempDir()}
	if opts.pkg, err = packageDir(c); err != nil {
		t.Fatal(err)
	}
	artifacts, err := buildPlatform(c, "windows", "amd64", opts)
	if err != nil {
		t.Fatal(err)
	}
	return dir, opts.work, artifacts
}

// shellFolder returns the path of a special folder of the user on
// Windows, as "Programs", the Start menu's, and "Desktop".
func shellFolder(t *testing.T, folder string) string {
	t.Helper()
	out, err := exec.Command("powershell", "-NoProfile", "-Command", "[Environment]::GetFolderPath('"+folder+"')").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// TestWindowsInstallerDisplayName builds the installer of an app that
// Windows shows by another name than its executable's, in Chinese and with
// a character file names cannot hold. On Windows, it installs the app over
// the shortcuts an earlier version made under the name, as stand-ins: the
// Start menu's goes and the desktop's is made again under the display
// name; Settings > Apps lists the app by it, and the uninstall removes the
// shortcuts.
func TestWindowsInstallerDisplayName(t *testing.T) {
	_, _, artifacts := buildTestInstaller(t, map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"mygo.json": `{"name": "Display Test", "identifier": "com.example.displaytest", "version": "1.0.0", "windows": {"displayName": "显示测试: Beta"}}`,
	})
	setup := artifacts[len(artifacts)-1]
	if filepath.Base(setup) != "display-test-setup-1.0.0-amd64.exe" {
		t.Fatalf("artifacts = %q", artifacts)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return
	}
	programs, desktop := shellFolder(t, "Programs"), shellFolder(t, "Desktop")
	oldStart, oldDesktop := filepath.Join(programs, "Display Test.lnk"), filepath.Join(desktop, "Display Test.lnk")
	startMenu, newDesktop := filepath.Join(programs, "显示测试- Beta.lnk"), filepath.Join(desktop, "显示测试- Beta.lnk")
	for _, f := range []string{oldStart, oldDesktop} {
		if err := os.WriteFile(f, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, f := range []string{oldStart, oldDesktop, startMenu, newDesktop} {
			os.Remove(f)
		}
	})
	install := filepath.Join(t.TempDir(), "DisplayTest")
	if out, err := exec.Command(setup, "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("installing: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(install, "Display Test.exe")); err != nil {
		t.Errorf("the executable is not named after the name: %v", err)
	}
	for _, f := range []string{startMenu, newDesktop} {
		if !fileExists(f) {
			t.Errorf("the installer made no %s", f)
		}
	}
	for _, f := range []string{oldStart, oldDesktop} {
		if fileExists(f) {
			t.Errorf("the installer left %s", f)
		}
	}
	// The display name goes through the environment, as the console's code
	// page could not carry it.
	check := exec.Command("powershell", "-NoProfile", "-Command",
		`$v = (Get-ItemProperty 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\com.example.displaytest').DisplayName; if ($v -ne $env:MYGO_WANT) { exit 1 }`)
	check.Env = append(os.Environ(), "MYGO_WANT=显示测试: Beta")
	if out, err := check.CombinedOutput(); err != nil {
		t.Errorf("Settings > Apps does not list the app as 显示测试: Beta: %v\n%s", err, out)
	}
	defer func() {
		for _, f := range []string{startMenu, newDesktop} {
			if fileExists(f) {
				t.Errorf("the uninstaller left %s", f)
			}
		}
	}()
	uninstall(t, install)
}

// TestNSISDisplayName checks the names in the installer's script: the
// shortcuts, the entry in Settings > Apps, the installer's title and its
// messages show windows.displayName, or else the name, while the
// executable and the install folder keep the name's.
func TestNSISDisplayName(t *testing.T) {
	for _, tc := range []struct {
		name, displayName string
		has, hasNot       []string
	}{{
		name: "My App",
		has: []string{
			"\nName \"My App\"\n",
			`InstallDir "$LOCALAPPDATA\Programs\My App"`,
			`CreateShortCut "$SMPROGRAMS\My App.lnk" "$INSTDIR\My App.exe"`,
			`CreateShortCut "$DESKTOP\My App.lnk" "$INSTDIR\My App.exe"`,
			`Push "$SMPROGRAMS\My App.lnk"`,
			`Push "$DESKTOP\My App.lnk"`,
			`"DisplayName" "My App"`,
			`"My App is running. Close it to continue?"`,
		},
		hasNot: []string{`Delete "$SMPROGRAMS`, `Delete "$DESKTOP`},
	}, {
		name: "weather", displayName: "天气助手",
		has: []string{
			"\nName \"天气助手\"\n",
			`InstallDir "$LOCALAPPDATA\Programs\weather"`,
			`CreateShortCut "$SMPROGRAMS\天气助手.lnk" "$INSTDIR\weather.exe"`,
			`CreateShortCut "$DESKTOP\天气助手.lnk" "$INSTDIR\weather.exe"`,
			`Push "$SMPROGRAMS\天气助手.lnk"`,
			`Push "$DESKTOP\天气助手.lnk"`,
			`"DisplayName" "天气助手"`,
			`"DisplayIcon" "$INSTDIR\weather.exe"`,
			`"天气助手 is running. Close it to continue?"`,
			`"URL:天气助手"`,
			// The shortcuts of an earlier version, named after the name.
			strings.Join([]string{
				`  Delete "$SMPROGRAMS\weather.lnk"`,
				`  IfFileExists "$DESKTOP\weather.lnk" 0 +3`,
				`  Delete "$DESKTOP\weather.lnk"`,
				`  Call CreateDesktopShortcut`,
				`  CreateShortCut "$SMPROGRAMS\天气助手.lnk"`,
			}, "\n"),
		},
		hasNot: []string{"\nName \"weather\"", `"DisplayName" "weather"`, `weather.lnk" "$INSTDIR`},
	}, {
		// Characters file names cannot hold become "-" in the shortcuts'
		// names only; NSIS's own characters are escaped everywhere, and
		// the title doubles the ampersand for the buttons' texts.
		name: "data-tool", displayName: `Data: "Tools" $1 & <More>?`,
		has: []string{
			"\nName " + `"Data: $\"Tools$\" $$1 & <More>?" "Data: $\"Tools$\" $$1 && <More>?"` + "\n",
			`InstallDir "$LOCALAPPDATA\Programs\data-tool"`,
			`CreateShortCut "$SMPROGRAMS\Data- -Tools- $$1 & -More--.lnk" "$INSTDIR\data-tool.exe"`,
			`CreateShortCut "$DESKTOP\Data- -Tools- $$1 & -More--.lnk" "$INSTDIR\data-tool.exe"`,
			`"DisplayName" "Data: $\"Tools$\" $$1 & <More>?"`,
			`Delete "$SMPROGRAMS\data-tool.lnk"`,
		},
	}, {
		// A name that differs from the name in case only names the same
		// files, which the installer keeps.
		name: "weather", displayName: " Weather ",
		has:    []string{"\nName \"Weather\"\n", `CreateShortCut "$SMPROGRAMS\Weather.lnk" "$INSTDIR\weather.exe"`, `"DisplayName" "Weather"`},
		hasNot: []string{`Delete "$SMPROGRAMS`},
	}} {
		c := &Config{Name: tc.name, Identifier: "com.example.app", Version: "1.0.0", URLSchemes: []string{"example"}, Windows: Windows{DisplayName: tc.displayName}}
		exe := c.executableName() + ".exe"
		script, err := installerScript(c, "", `C:\out\setup.exe`, exe, "  File \"C:\\stage\\"+exe+"\"\n", "", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range tc.has {
			if !strings.Contains(script, s) {
				t.Errorf("%q: no %q in:\n%s", tc.displayName, s, script)
			}
		}
		for _, s := range tc.hasNot {
			if strings.Contains(script, s) {
				t.Errorf("%q: %q in:\n%s", tc.displayName, s, script)
			}
		}
		if !strings.HasPrefix(script, "Unicode true\n") {
			t.Errorf("the script does not start with Unicode true:\n%s", script)
		}
		if tool := makensis(); tool != "" && !testing.Short() {
			// makensis takes the script, with a stand-in for the app.
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, exe), []byte("MZ"), 0o644)
			out := filepath.Join(dir, "setup.exe")
			script, err := installerScript(c, "", out, exe, "  File "+nsisString(filepath.Join(dir, exe))+"\n", "", "", nil)
			if err != nil {
				t.Fatal(err)
			}
			nsi := filepath.Join(dir, "installer.nsi")
			if err := writeNSIS(nsi, script); err != nil {
				t.Fatal(err)
			}
			if b, err := exec.Command(tool, "-V2", nsi).CombinedOutput(); err != nil {
				t.Errorf("%q: makensis: %v\n%s", tc.displayName, err, b)
			}
		}
	}
}

// TestNSISInstallDirName checks windows.installDirName: it names the
// install folder, and nothing else, while the executable, the shortcuts,
// the entry in Settings > Apps and the messages keep the name; the
// installer of a later version installs where the app is. makensis takes
// InstallDirRegKey in TestNSISDisplayName's scripts.
func TestNSISInstallDirName(t *testing.T) {
	key := `Software\Microsoft\Windows\CurrentVersion\Uninstall\com.example.zhilian`
	for _, tc := range []struct {
		installDirName, dir string
	}{
		{"", "智链"},
		{"  ", "智链"},
		{" ZhiLian ", "ZhiLian"},
		{"Wen Jing $1", "Wen Jing $$1"},
	} {
		c := &Config{Name: "智链", Identifier: "com.example.zhilian", Version: "1.0.0", Windows: Windows{InstallDirName: tc.installDirName}}
		exe := c.executableName() + ".exe"
		script, err := installerScript(c, "", `C:\out\setup.exe`, exe, "  File \"C:\\stage\\"+exe+"\"\n", "", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []string{
			`InstallDir "$LOCALAPPDATA\Programs\` + tc.dir + `"` + "\n",
			`InstallDirRegKey HKCU "` + key + `" "InstallLocation"` + "\n",
			"\nName \"智链\"\n",
			`CreateShortCut "$SMPROGRAMS\智链.lnk" "$INSTDIR\智链.exe"`,
			`CreateShortCut "$DESKTOP\智链.lnk" "$INSTDIR\智链.exe"`,
			`"DisplayName" "智链"`,
			`"DisplayIcon" "$INSTDIR\智链.exe"`,
			`"InstallLocation" "$INSTDIR"`,
			`"智链 is running. Close it to continue?"`,
			`RMDir "$INSTDIR"`,
		} {
			if !strings.Contains(script, s) {
				t.Errorf("%q: no %q in:\n%s", tc.installDirName, s, script)
			}
		}
		if tc.dir != "智链" && strings.Contains(script, `Programs\智链`) {
			t.Errorf("%q: the script names the folder after the name:\n%s", tc.installDirName, script)
		}
	}
}

// TestNSISSourceLanguage checks that the installer of an app written in
// Chinese alone is in Chinese only, and that of an app with more languages
// in each of them, the app's own first, which it falls back to.
func TestNSISSourceLanguage(t *testing.T) {
	for _, tc := range []struct {
		catalogs map[string]string
		want     string
	}{
		{map[string]string{"zh-Hans.json": "{}"}, "!insertmacro MUI_LANGUAGE \"SimpChinese\"\n"},
		{map[string]string{"zh-Hans.json": "{}", "en.json": "{}"}, "!insertmacro MUI_LANGUAGE \"SimpChinese\"\n!insertmacro MUI_LANGUAGE \"English\"\n"},
	} {
		c := project(t, `{"name": "智链", "identifier": "com.example.zhilian", "version": "1.0.0", "i18n": {"source": "zh-Hans"}}`, tc.catalogs)
		script, err := installerScript(c, "", `C:\out\setup.exe`, "智链.exe", "", "", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		var got string
		for _, line := range strings.SplitAfter(script, "\n") {
			if strings.HasPrefix(line, "!insertmacro MUI_LANGUAGE ") {
				got += line
			}
		}
		if got != tc.want {
			t.Errorf("with %v, the installer's languages are\n%swant\n%s", tc.catalogs, got, tc.want)
		}
	}
}

// TestNSISHooks checks windows.installerInclude: the installer includes the
// script after MUI2.nsh and the languages, and runs its customInit,
// customInstall and customUnInstall macros, each only when the script
// defines it; without it, the script has none of that. makensis, when
// there is one, takes a script that defines some of the macros only.
func TestNSISHooks(t *testing.T) {
	dir := t.TempDir()
	nsh := filepath.Join(dir, "build", "installer.nsh")
	if err := os.MkdirAll(filepath.Dir(nsh), 0o755); err != nil {
		t.Fatal(err)
	}
	hooks := "!macro customInstall\n  CreateDirectory \"$LOCALAPPDATA\\Hooks Test\"\n!macroend\n" +
		"!macro customUnInstall\n  RMDir \"$LOCALAPPDATA\\Hooks Test\"\n!macroend\n"
	if err := os.WriteFile(nsh, []byte(hooks), 0o644); err != nil {
		t.Fatal(err)
	}
	c := &Config{Name: "Hooks Test", Identifier: "com.example.hooks", Version: "1.0.0", root: dir}
	exe := c.executableName() + ".exe"
	files := "  File \"C:\\stage\\" + exe + "\"\n"
	plain, err := installerScript(c, "", `C:\out\setup.exe`, exe, files, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	// .onInit always takes the installer's mutex.
	for _, s := range []string{"!ifmacrodef", "customInit", "customInstall", "customUnInstall", "installer.nsh"} {
		if strings.Contains(plain, s) {
			t.Errorf("without windows.installerInclude, %q in:\n%s", s, plain)
		}
	}

	c.Windows.InstallerInclude = "build/installer.nsh"
	script, err := installerScript(c, "", `C:\out\setup.exe`, exe, files, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	include := "!include " + nsisString(nsh) + "\n"
	install := "  !ifmacrodef customInstall\n    !insertmacro customInstall\n  !endif\n"
	uninstall := "  !ifmacrodef customUnInstall\n    !insertmacro customUnInstall\n  !endif\n"
	// Each in its place: the include after MUI2.nsh and the languages,
	// customInstall last in the install's section, and customUnInstall in
	// the uninstall's, before it removes $INSTDIR.
	for _, order := range [][]string{
		{`!include "MUI2.nsh"`, `!insertmacro MUI_LANGUAGE "English"`, include, "\nSection\n"},
		{include, "Function .onInit\n", "  !ifmacrodef customInit\n    !insertmacro customInit\n  !endif\n  Return\n"},
		{"\nSection\n", "  WriteUninstaller ", install + "SectionEnd\n"},
		{"Section \"Uninstall\"\n", "  DeleteRegKey HKCU ", uninstall, "  RMDir \"$INSTDIR\"\n"},
	} {
		at := 0
		for _, s := range order {
			i := strings.Index(script[at:], s)
			if i < 0 {
				t.Fatalf("no %q after %q in:\n%s", s, order[0], script)
			}
			at += i + len(s)
		}
	}
	// It adds them, and changes nothing else.
	rest := script
	initHook := "  !ifmacrodef customInit\n    !insertmacro customInit\n  !endif\n"
	for _, s := range []string{include, initHook, install, uninstall} {
		if strings.Count(rest, s) != 1 {
			t.Fatalf("%q is not in the script once:\n%s", s, script)
		}
		rest = strings.Replace(rest, s, "", 1)
	}
	if rest != plain {
		t.Errorf("windows.installerInclude changed more than the hooks:\n%s", script)
	}

	for _, missing := range []string{"build/missing.nsh", "build"} {
		c.Windows.InstallerInclude = missing
		if _, err := installerScript(c, "", `C:\out\setup.exe`, exe, files, "", "", nil); err == nil || !strings.Contains(err.Error(), "windows.installerInclude") {
			t.Errorf("windows.installerInclude %q: %v", missing, err)
		}
	}

	if tool := makensis(); tool != "" && !testing.Short() {
		c.Windows.InstallerInclude = "build/installer.nsh"
		os.WriteFile(filepath.Join(dir, exe), []byte("MZ"), 0o644)
		out := filepath.Join(dir, "setup.exe")
		script, err := installerScript(c, "", out, exe, "  File "+nsisString(filepath.Join(dir, exe))+"\n", "", "", nil)
		if err != nil {
			t.Fatal(err)
		}
		nsi := filepath.Join(dir, "installer.nsi")
		if err := writeNSIS(nsi, script); err != nil {
			t.Fatal(err)
		}
		if b, err := exec.Command(tool, "-V2", nsi).CombinedOutput(); err != nil {
			t.Errorf("makensis: %v\n%s", err, b)
		}
	}
}

// TestWindowsInstallerLanguages compiles an installer in two languages and
// checks that makensis warns of nothing: it reports a LangString missing in
// a language only as a warning, which does not fail the build.
func TestWindowsInstallerLanguages(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles an installer")
	}
	tool, err := nsisCompiler()
	if err != nil {
		t.Fatal(err)
	} else if tool == "" {
		t.Skip("NSIS is not installed")
	}
	c := project(t, `{"name": "Notes", "identifier": "com.example.notes", "version": "1.0.0", "i18n": {}}`, map[string]string{
		"en.json":      `{}`,
		"zh-Hans.json": `{}`,
	})
	stage, work := t.TempDir(), t.TempDir()
	writeFiles(t, stage, map[string]string{"Notes.exe": "app"})
	if _, err := writeInstaller(c, stage, work, "Notes.exe", "amd64", []string{"Notes.exe"}); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(tool, "-V2", filepath.Join(work, "installer.nsi")).CombinedOutput(); err != nil || strings.Contains(strings.ToLower(string(out)), "warning") {
		t.Fatalf("makensis must compile every language without warnings: %v\n%s", err, out)
	}
}

// TestWindowsInstallerEncoding checks that installer.nsi is written as UTF-8
// with a byte order mark. NSIS on Windows reads a script without a BOM as
// ANSI, in the machine's code page, even after `Unicode true`: a name or a
// path with a non-ASCII character, e.g. 望言字幕导出 or a folder named 中文,
// then makes makensis report "no files found" and the build fails.
func TestWindowsInstallerEncoding(t *testing.T) {
	dir := testModule(t, map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"mygo.json": `{"name": "望言字幕导出", "identifier": "com.example.encoding", "version": "1.0.0"}`,
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "望言字幕导出" {
		t.Fatalf("the name is %q", c.Name)
	}
	nsi := filepath.Join(t.TempDir(), "installer.nsi")
	// The name reaches the script, as it does in writeInstaller.
	script := "Unicode true\nName " + nsisString(c.Name) + "\n"
	if err := writeNSIS(nsi, script); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(nsi)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 3 || b[0] != 0xEF || b[1] != 0xBB || b[2] != 0xBF {
		t.Fatalf("installer.nsi starts with % x, want a UTF-8 byte order mark", b[:min(len(b), 3)])
	}
	if !strings.Contains(string(b[3:]), "望言字幕导出") {
		t.Fatalf("installer.nsi does not hold the name as UTF-8: %q", b)
	}
}

// uninstall runs the uninstaller of the app installed in dir silently, and
// waits until it is done.
func uninstall(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.Command(filepath.Join(dir, "Uninstall.exe"), "/S").CombinedOutput(); err != nil {
		t.Fatalf("uninstalling: %v\n%s", err, out)
	}
	// The uninstaller copies itself away and runs from there, and removes
	// the app's folder last.
	deadline := time.Now().Add(30 * time.Second)
	for {
		// It removes the files it installed and keeps any it did not.
		if _, err := os.Stat(filepath.Join(dir, "Uninstall.exe")); os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the uninstaller left its executable installed")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestWindowsSigning signs a build with a throwaway self-signed
// certificate. It runs on CI only (MYGO_TEST_SIGN), as it adds the
// certificate to the user's store for a moment.
func TestWindowsSigning(t *testing.T) {
	if runtime.GOOS != "windows" || os.Getenv("MYGO_TEST_SIGN") == "" {
		t.Skip("set MYGO_TEST_SIGN on a disposable Windows machine")
	}
	pfx := filepath.Join(t.TempDir(), "test.pfx")
	ps := func(script string) string {
		t.Helper()
		out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
		if err != nil {
			t.Fatalf("powershell: %v\n%s", err, out)
		}
		return strings.TrimSpace(string(out))
	}
	ps(`$c = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=MyGo Test" -CertStoreLocation Cert:\CurrentUser\My; ` +
		`$p = ConvertTo-SecureString -String "secret" -Force -AsPlainText; ` +
		`Export-PfxCertificate -Cert $c -FilePath "` + pfx + `" -Password $p | Out-Null; Remove-Item $c.PSPath`)
	t.Setenv("MYGO_WINDOWS_CERTIFICATE_PASSWORD", "secret")
	dir := testModule(t, map[string]string{
		"main.go":   "package main\n\nfunc main() {}\n",
		"mygo.json": `{"name": "Signed App", "version": "1.0.0", "windows": {"certificate": "` + filepath.ToSlash(pfx) + `"}}`,
	})
	c, err := loadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	opts := buildOptions{sign: "-", work: t.TempDir()}
	if opts.pkg, err = packageDir(c); err != nil {
		t.Fatal(err)
	}
	artifacts, err := buildPlatform(c, "windows", "amd64", opts)
	if err != nil {
		t.Fatal(err)
	}
	signer := func(file string) string {
		return ps(`(Get-AuthenticodeSignature "` + file + `").SignerCertificate.Subject`)
	}
	for _, a := range artifacts {
		if strings.HasSuffix(a, ".exe") {
			if subject := signer(a); subject != "CN=MyGo Test" {
				t.Errorf("%s is signed by %q", filepath.Base(a), subject)
			}
		}
	}
	// The installer writes an uninstaller signed with the app.
	install := filepath.Join(t.TempDir(), "SignedApp")
	if out, err := exec.Command(artifacts[len(artifacts)-1], "/S", "/D="+install).CombinedOutput(); err != nil {
		t.Fatalf("installing: %v\n%s", err, out)
	}
	if subject := signer(filepath.Join(install, "Uninstall.exe")); subject != "CN=MyGo Test" {
		t.Errorf("Uninstall.exe is signed by %q", subject)
	}
	uninstall(t, install)
}

func TestNotificationInstallerLifecycle(t *testing.T) {
	c := &Config{Windows: Windows{Notifications: true}}
	install, uninstall := nsisNotifications(c, "App With Spaces.exe")
	for _, want := range []string{
		`ExecWait '"$INSTDIR\App With Spaces.exe" --mygo-register-notifications`,
		`IfErrors`,
		`IntCmp $0 0`,
		`SetErrorLevel 1`,
		`Abort "Native notification registration failed. Close the application and retry setup."`,
	} {
		if !strings.Contains(install, want) {
			t.Errorf("missing %q in install script:\n%s", want, install)
		}
	}
	exec := `ExecWait '"$INSTDIR\App With Spaces.exe" --mygo-unregister-notifications'`
	if !strings.Contains(uninstall, exec) {
		t.Fatalf("uninstall does not run the helper:\n%s", uninstall)
	}
	if strings.Index(uninstall, exec) > strings.LastIndex(uninstall, "ClearErrors") {
		t.Fatalf("uninstall must ClearErrors after the helper:\n%s", uninstall)
	}
	for _, forbidden := range []string{"SetErrorLevel", "Abort", "IfErrors", "IntCmp"} {
		if strings.Contains(uninstall, forbidden) {
			t.Errorf("uninstall must continue after helper failure, found %q in:\n%s", forbidden, uninstall)
		}
	}
	c.Windows.Notifications = false
	if a, b := nsisNotifications(c, "App.exe"); a != "" || b != "" {
		t.Fatal("registered without opt in")
	}
}
