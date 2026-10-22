package macos

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"arion.local/core/domain"
)

func bundleFixture(t *testing.T, root, name, contents string) string {
	t.Helper()
	bundle := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(bundle, "Contents", "MacOS"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "Contents", "MacOS", "ArionInput"), []byte(contents), 0755); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func bundleContents(t *testing.T, bundle string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(bundle, "Contents", "MacOS", "ArionInput"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func requireNoTransaction(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".arion-input-install-") {
			t.Fatalf("transaction directory left after cleanup: %s", entry.Name())
		}
	}
}

func TestInputBundleStagingFailurePreservesInstalledBytes(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	called := false
	err := replaceInputBundle(filepath.Join(root, "missing.app"), old, func() error { called = true; return nil }, func() error { called = true; return nil })
	if err == nil || called || bundleContents(t, old) != "previous" {
		t.Fatalf("missing payload changed installed source: %v", err)
	}
	new := bundleFixture(t, root, "source.app", "new")
	if err := syscall.Mkfifo(filepath.Join(new, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	err = replaceInputBundle(new, old, func() error { called = true; return nil }, func() error { called = true; return nil })
	if err == nil || called || bundleContents(t, old) != "previous" {
		t.Fatalf("unsupported payload entry changed installed source: %v", err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleRegistrationFailureRestoresPreviousBytes(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	failure := errors.New("registration refused")
	restored := false
	err := replaceInputBundle(new, old, func() error {
		if bundleContents(t, old) != "new" {
			t.Fatal("registration did not receive staged bundle")
		}
		return failure
	}, func() error {
		restored = true
		if bundleContents(t, old) != "previous" {
			t.Fatal("restoration callback saw wrong bytes")
		}
		return nil
	})
	if !errors.Is(err, failure) || !restored || bundleContents(t, old) != "previous" {
		t.Fatalf("failed update not rolled back: %v", err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleFirstInstallFailureRemovesFailedPayload(t *testing.T) {
	root := t.TempDir()
	new := bundleFixture(t, root, "source.app", "new")
	destination := filepath.Join(root, "ArionInput.app")
	failure, cleanup := errors.New("enable refused"), errors.New("disable refused")
	err := replaceInputBundle(new, destination, func() error { return failure }, func() error {
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatalf("failed first install remains: %v", err)
		}
		return cleanup
	})
	if !errors.Is(err, failure) || !errors.Is(err, cleanup) {
		t.Fatalf("registration or cleanup failure lost: %v", err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleReplacementPreservesExecutableAndSymlink(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	if err := os.Symlink("MacOS/ArionInput", filepath.Join(new, "Contents", "current")); err != nil {
		t.Fatal(err)
	}
	err := replaceInputBundle(new, old, func() error { return nil }, func() error {
		t.Fatal("successful replacement invoked restoration")
		return nil
	})
	if err != nil || bundleContents(t, old) != "new" {
		t.Fatalf("replacement failed: %v", err)
	}
	info, err := os.Stat(filepath.Join(old, "Contents", "MacOS", "ArionInput"))
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("executable mode changed: %v", err)
	}
	link, err := os.Readlink(filepath.Join(old, "Contents", "current"))
	if err != nil || link != "MacOS/ArionInput" {
		t.Fatalf("bundle symlink changed: %q %v", link, err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleFailedRollbackRetainsRecoveryCopy(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	failure := errors.New("registration refused")
	err := replaceInputBundle(new, old, func() error {
		// Simulate a filesystem mutation making rollback impossible.
		if err := os.RemoveAll(old); err != nil {
			t.Fatal(err)
		}
		return failure
	}, func() error {
		t.Fatal("registration restoration ran without restored bytes")
		return nil
	})
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "recovery files retained") {
		t.Fatalf("rollback failure not reported: %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(root, ".arion-input-install-*", "previous.app"))
	if err != nil || len(backups) != 1 || bundleContents(t, backups[0]) != "previous" {
		t.Fatalf("sole previous payload was discarded: %v %v", backups, err)
	}
}

func TestInputBundleRemovalStopsOnReleaseFailure(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	failure := errors.New("source still selected")
	if err := removeInputBundle(old, func() error { return failure }); !errors.Is(err, failure) || bundleContents(t, old) != "previous" {
		t.Fatalf("release failure deleted installed bundle: %v", err)
	}
	if err := removeInputBundle(old, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("released bundle not removed: %v", err)
	}
}

func TestInputBundleIdentityReadsCurrentMetadata(t *testing.T) {
	bundle := bundleFixture(t, t.TempDir(), "ArionInput.app", "fixture")
	metadata := filepath.Join(bundle, "Contents", "Info.plist")
	for _, test := range []struct {
		name, body string
		match      bool
	}{
		{"missing", "", false},
		{"malformed", "not a property list", false},
		{"array", "<array/>", false},
		{"wrong component type", "<dict><key>ComponentInputModeDict</key><array/></dict>", false},
		{"current", "<dict><key>ComponentInputModeDict</key><dict><key>tsInputModeListKey</key><dict><key>" + IMESourceID + "</key><dict><key>TISInputSourceID</key><string>" + IMESourceID + "</string></dict></dict></dict></dict>", true},
		{"legacy replacement", "<dict><key>CFBundleIdentifier</key><string>local.arion.inputmethod</string></dict>", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.body != "" {
				data := "<?xml version=\"1.0\"?><plist version=\"1.0\">" + test.body + "</plist>"
				if err := os.WriteFile(metadata, []byte(data), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if got := inputBundleMatches(bundle); got != test.match {
				t.Fatalf("metadata match = %v, want %v", got, test.match)
			}
		})
	}
}

func TestInputBundlePendingEnablementRetainsRegisteredBundle(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "registered")
	current := IMEAbsent
	err := replaceInputBundle(new, old, func() error {
		return registerAndEnableInputSource(func() error {
			if bundleContents(t, old) != "registered" {
				t.Fatal("registered wrong installed bytes")
			}
			current = IMERegistered
			return nil
		}, func() int { return 0 }, func() IMEState { return current }, false)
	}, func() error { t.Fatal("pending registration must not roll back"); return nil })
	if err != nil || current != IMERegistered || bundleContents(t, old) != "registered" {
		t.Fatalf("pending input source was removed or marked enabled: state=%v error=%v", current, err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleRegistrationCompletionRejectsRealFailures(t *testing.T) {
	for _, test := range []struct {
		name         string
		registration error
		registered   IMEState
		enable       int
		final        IMEState
	}{
		{"registration refused", errors.New("registration refused"), IMEAbsent, 0, IMEAbsent},
		{"enable refused", nil, IMERegistered, -50, IMERegistered},
		{"disappeared after enable", nil, IMERegistered, 0, IMEAbsent},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			old := bundleFixture(t, root, "ArionInput.app", "previous")
			new := bundleFixture(t, root, "source.app", "new")
			current := test.registered
			restored := false
			err := replaceInputBundle(new, old, func() error {
				return registerAndEnableInputSource(func() error { return test.registration }, func() int {
					current = test.final
					return test.enable
				}, func() IMEState { return current }, false)
			}, func() error { restored = true; return nil })
			if err == nil || !restored || bundleContents(t, old) != "previous" {
				t.Fatalf("genuine failure retained invalid install: %v restored=%v", err, restored)
			}
			requireNoTransaction(t, root)
		})
	}
}

func TestInputBundleRestoreRequiresPreviouslyEnabledState(t *testing.T) {
	if err := enableInputSource(func() int { return 0 }, func() IMEState { return IMERegistered }, true); err == nil {
		t.Fatal("restoration silently downgraded enabled source to pending")
	}
}

func inputPayloadMetadata(t *testing.T, bundle string) {
	t.Helper()
	body := `<?xml version="1.0"?><plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>si.astrafoundry.inputmethod.arion</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleExecutable</key><string>ArionInput</string>
<key>ComponentInputModeDict</key><dict><key>tsInputModeListKey</key><dict><key>` + IMESourceID + `</key><dict><key>TISInputSourceID</key><string>` + IMESourceID + `</string></dict></dict></dict></dict></plist>`
	if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestInputBundleCopiedStateIsSeparateFromNativeRegistration(t *testing.T) {
	bundle := bundleFixture(t, t.TempDir(), "ArionInput.app", "fixture")
	inputPayloadMetadata(t, bundle)
	if got := queryInstalledIME(bundle, func() IMEState { return IMEAbsent }); got != IMECopied {
		t.Fatalf("copied bundle misreported as absent: %v", got)
	}
	for _, state := range []IMEState{IMERegistered, IMEEnabled, IMESelected} {
		if got := queryInstalledIME(bundle, func() IMEState { return state }); got != state {
			t.Fatalf("native state changed: got %v want %v", got, state)
		}
	}
	if err := os.Remove(filepath.Join(bundle, "Contents", "Info.plist")); err != nil {
		t.Fatal(err)
	}
	if got := queryInstalledIME(bundle, func() IMEState { t.Fatal("invalid payload queried live sources"); return IMESelected }); got != IMEAbsent {
		t.Fatalf("invalid copied bundle treated as pending: %v", got)
	}
}

func TestInputBundlePayloadRejectsUnsignedAndAdHocSignatures(t *testing.T) {
	for _, adHoc := range []bool{false, true} {
		name := "unsigned"
		if adHoc {
			name = "ad hoc"
		}
		t.Run(name, func(t *testing.T) {
			bundle := bundleFixture(t, t.TempDir(), "ArionInput.app", "fixture")
			inputPayloadMetadata(t, bundle)
			if adHoc {
				binary, err := os.ReadFile("/usr/bin/true")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(bundle, "Contents", "MacOS", "ArionInput"), binary, 0755); err != nil {
					t.Fatal(err)
				}
				if output, err := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--identifier", "si.astrafoundry.inputmethod.arion", bundle).CombinedOutput(); err != nil {
					t.Fatalf("sign isolated ad hoc fixture: %v %s", err, output)
				}
			}
			if !inputBundleMatches(bundle) {
				t.Fatal("fixture must reach signature validation")
			}
			if err := ValidateIMEPayload(bundle); domain.CodeOf(err) != domain.CodeArtifactUnverified {
				t.Fatalf("untrusted payload accepted or wrong failure category: %v", err)
			}
		})
	}
}

// Optional release-artifact check; reads only the explicitly supplied bundle.
// It never queries, installs, registers, enables or selects an input source.
func TestInputBundleConfiguredDeveloperIDPayload(t *testing.T) {
	bundle := os.Getenv("ARION_TEST_IME_PAYLOAD")
	if bundle == "" {
		t.Skip("set ARION_TEST_IME_PAYLOAD to a completed profile-bearing Developer ID host")
	}
	if !filepath.IsAbs(bundle) {
		t.Fatal("ARION_TEST_IME_PAYLOAD must be an absolute bundle path")
	}
	if err := ValidateIMEPayload(bundle); err != nil {
		t.Fatalf("configured release payload did not pass runtime preflight: %v", err)
	}
}

func TestBundleDiffers(t *testing.T) {
	write := func(root, build string) {
		for _, name := range []string{"Contents/MacOS/ArionInput", "Contents/_CodeSignature/CodeResources", "Contents/Info.plist"} {
			path := filepath.Join(root, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(build+name), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	dir := t.TempDir()
	installed, bundled := filepath.Join(dir, "installed.app"), filepath.Join(dir, "bundled.app")
	write(bundled, "new")
	if bundleDiffers(installed, bundled) {
		t.Fatal("absent installation reported as an update")
	}
	write(installed, "new")
	if bundleDiffers(installed, bundled) {
		t.Fatal("identical build reported as an update")
	}
	if err := os.WriteFile(filepath.Join(installed, "Contents/_CodeSignature/CodeResources"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !bundleDiffers(installed, bundled) {
		t.Fatal("changed resources not reported")
	}
	write(installed, "old")
	if !bundleDiffers(installed, bundled) {
		t.Fatal("older build not reported")
	}
}

func TestInputBundleHandoverPrecedesReplacement(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	stopped := false
	err := replaceInputBundleWithLifecycle(new, old, func() error {
		if !stopped || bundleContents(t, old) != "new" {
			t.Fatal("activated before old host stopped or new bytes promoted")
		}
		return nil
	}, func() error { t.Fatal("successful handover restored old host"); return nil }, inputBundleLifecycle{beforeReplace: func() error {
		if bundleContents(t, old) != "previous" {
			t.Fatal("old bytes changed before process exit")
		}
		// Staging must have completed before the running host is interrupted.
		staged, err := filepath.Glob(filepath.Join(root, ".arion-input-install-*", "new.app", "Contents", "MacOS", "ArionInput"))
		if err != nil || len(staged) != 1 {
			t.Fatal("host stopped before staging completed", err)
		}
		stopped = true
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleStopFailurePreservesOldBytesAndRestoresState(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	stopError := errors.New("old process still running")
	restored := 0
	err := replaceInputBundleWithLifecycle(new, old, func() error { t.Fatal("activated after stop failure"); return nil }, func() error { restored++; return nil }, inputBundleLifecycle{beforeReplace: func() error { return stopError }})
	if !errors.Is(err, stopError) || restored != 1 || bundleContents(t, old) != "previous" {
		t.Fatal("stop failure changed bytes or lost restoration", err, restored)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleReadinessFailureStopsNewHostBeforeRollback(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	failure := errors.New("new build did not connect")
	newStopped, restored := false, false
	err := replaceInputBundleWithLifecycle(new, old, func() error { return failure }, func() error {
		if !newStopped || bundleContents(t, old) != "previous" {
			t.Fatal("restored before failed host exit or old-byte restoration")
		}
		restored = true
		return nil
	}, inputBundleLifecycle{beforeReplace: func() error { return nil }, beforeRollback: func() error {
		if bundleContents(t, old) != "new" {
			t.Fatal("rolled bytes back while new host was live")
		}
		newStopped = true
		return nil
	}})
	if !errors.Is(err, failure) || !restored {
		t.Fatal("readiness rollback failed", err)
	}
	requireNoTransaction(t, root)
}

func TestInputBundleRollbackStopFailurePreservesBothBuilds(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	new := bundleFixture(t, root, "source.app", "new")
	failure := errors.New("handshake failed")
	stopError := errors.New("new host cannot stop")
	err := replaceInputBundleWithLifecycle(new, old, func() error { return failure }, func() error { t.Fatal("restored over running new host"); return nil }, inputBundleLifecycle{beforeReplace: func() error { return nil }, beforeRollback: func() error { return stopError }})
	backups, _ := filepath.Glob(filepath.Join(root, ".arion-input-install-*", "previous.app"))
	if !errors.Is(err, failure) || !errors.Is(err, stopError) || len(backups) != 1 || bundleContents(t, old) != "new" || bundleContents(t, backups[0]) != "previous" {
		t.Fatal("failed rollback lost recovery build", err)
	}
}

func TestProcessExitWaitCannotReportLiveHostAsStopped(t *testing.T) {
	if awaitInputMethodExit(func() int { return 1 }, 0) {
		t.Fatal("reported live process as stopped")
	}
	if !awaitInputMethodExit(func() int { return 0 }, 0) {
		t.Fatal("did not observe process exit")
	}
}

func TestInputBundlePendingDiscoveryRetainsBytesWithoutEnableRequest(t *testing.T) {
	root := t.TempDir()
	old := bundleFixture(t, root, "ArionInput.app", "previous")
	source := bundleFixture(t, root, "source.app", "new")
	err := replaceInputBundle(source, old, func() error {
		return registerAndEnableInputSource(func() error { return nil }, func() int { t.Fatal("enabled undiscovered source"); return -1 }, func() IMEState { return IMEAbsent }, false)
	}, func() error { t.Fatal("discovery delay rolled back validated bytes"); return nil })
	if err != nil || bundleContents(t, old) != "new" {
		t.Fatal("pending discovery lost bytes", err)
	}
	requireNoTransaction(t, root)
	if err := registerAndEnableInputSource(func() error { return nil }, func() int { return 0 }, func() IMEState { return IMEAbsent }, true); err == nil {
		t.Fatal("upgrade restored selected source as undiscovered")
	}
}
