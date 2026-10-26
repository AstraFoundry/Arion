# Windows input method

## Shared Pinyin work in progress

The shared Pinyin implementation uses the Windows native host. Historical voice-only behavior is distinct from the installed-editor results below. The build binds `keyboard/runtime.json` to the service core and maintenance helper through `ARION_KEYBOARD_MANIFEST_SHA256`; the stable loader remains independent. Changing any runtime member changes the core's build identity. Installation verifies the complete closure, copies it under the protected hash-versioned `keyboard` directory, applies Administrator/SYSTEM ownership and read/execute-only user ACLs to copied files, then revalidates before registration. Unknown members, case aliases, reparse points and hash mismatches are rejected.

Removal reserves delete access to every verified runtime file before unregistering, so a loaded module refuses removal while registration remains intact. It deletes only enumerated owned members and empty directories. Legacy voice-only versions remain removable. The native synthetic filesystem fixture covers locked/tampered/extra-file refusal and complete/legacy cleanup; it does not prove actual UAC installation.

`node scripts/test-windows-keyboard-module.mjs` builds and dynamically loads the production keyboard module with restricted DLL search, tests real conversion, shared-client reference counting, profile mismatch refusal, serialized host-thread calls and encrypted-profile reopen. It also runs the installer filesystem fixture. Reports record source and binary hashes under `.workspace/tests/windows-keyboard-module/`. These tests neither register a TIP nor establish installed-editor behavior. Multi-editor learning storage and macOS native regression remain required gates.

Reset invalidates existing sessions. If another editor holds the vault lease, reset refuses before deleting any data and reopens the original runtime only after authenticating the intact vault. The error remains visible, but subsequent typing and confirmed learning remain available. Other reset failures and failed validation stay unavailable; a new client cannot reuse that finalized runtime. The module fixture tests a separate process holding the lease, continued conversion and preserved ranking after refusal, followed by successful reset after peer exit. A tampered-vault case verifies that recovery is refused.

The current worktree builds a native x64 TSF keyboard and voice core, `ArionTSF.dll`, separately from MyGo. Its profile is **Arion**, Simplified Chinese (`0x0804`). Shared candidate and learning preferences are exposed on Windows. First installation/removal request administrator approval. Application updates maintain an existing enabled source: stable-loader installations can publish a core while selected; direct legacy registrations defer migration until unused. Absent or deliberately disabled sources are not installed/enabled from polling. The previously installed voice-only build used **Arion Voice**, English (United States, `0x0409`). Query identifies it explicitly and the versioned update transaction migrates it without separate uninstall/reinstall steps. Protected machine updates still require Windows administrator approval; denied authorization stops automatic retries for this app run.

## Supported boundary

Voice composition supports ordinary, medium-integrity x64 desktop editors with real TSF text stores. Elevated/high-integrity processes, low-integrity/AppContainer processes, 32-bit editors, password fields and read-only fields are unsupported. A selected source that refuses composition fails before capture; it never silently switches to clipboard or keystrokes. Stock `RICHEDIT50W` uses its native password/read-only styles as a verified safety adapter; custom stores must expose usable input-scope properties. Chromium marks its ordinary editable TSF stores `TS_SS_TRANSITORY | TS_SS_NOHIDDENTEXT`; transitory stores are accepted only with an explicit non-password input scope, writable status and successful composition acquisition. Unknown-scope compatibility contexts remain refused. See [Chromium's TSF text-store implementation](https://chromium.googlesource.com/chromium/src/+/main/ui/base/ime/win/tsf_text_store.cc).

The host observes TSF thread focus through `IsThreadFocus`, thread-focus notifications and keyboard-focus receipt. It must not infer the editing thread from the foreground top-level window: WebView2 can place its focused text store in another process. Keyboard-focus receipt refreshes the current TSF document even when no subsequent thread-focus transition occurs. Keyboard-focus loss and thread-focus loss share idempotent draft cleanup and voice revocation; neither requires the other callback. The owned installed-browser regression distinguishes first-key behavior immediately after source selection from input that starts working only after focus leaves and returns.

Candidate layout is observed separately from composition writes. The host subscribes to `ITfTextLayoutSink` and requests a coalesced asynchronous read edit session after layout changes. A browser can temporarily return `TS_E_NOLAYOUT` for the newly written caret; the panel remains hidden until a fresh, unclipped current rectangle is available, without reusing its old coordinates. Layout reads recheck safety, exact owned text/caret and composition revision and do not process engine keys or alter the document. The native ACP gate tests delayed geometry and read-lock-only recovery; installed-browser acceptance additionally checks candidate visibility after every letter, rather than treating a successful final commit as proof of visibility.

Chromium exposes password inputs as `IS_PRIVATE`, not only `IS_PASSWORD`.
Both scopes are refused before Pinyin or voice composition acquisition. Chromium
also uses `IS_PRIVATE` for incognito/guest fields; those fields are consequently
unsupported by this conservative adapter. This avoids treating a private scope
as an ordinary writable field. See the primary
[Chromium input-scope mapping](https://chromium.googlesource.com/chromium/src/+/main/ui/base/ime/win/tsf_input_scope.cc).

The DLL uses its editor's COM apartment, focus/context notifications and TSF edit sessions. Bounded JSON lines cross `\\.\pipe\Arion.Voice.TSF.<current-user-SID>`. The listener has a protected current-user DACL, reserves the first instance and rejects remote clients. The host verifies server user SID, Windows session and ordinary integrity. Overlapped I/O workers have bounded queues/deadlines and never touch COM objects. Native incoming/disconnect notifications carry an immutable connection generation, so an old message already held by an outer message pump cannot affect a replacement connection after reentrant focus changes.

Activation, session and revision bind composition ownership. The application verifies the pipe client's actual process identity and loaded protected TSF module, then checks its supported wire protocol. A compatible retained component need not equal the current application's bundled DLL hash. Windows protocol 2 requires the complete verified keyboard closure and refusal of private/password/PIN scopes; legacy protocols 0 and 1 are not accepted for voice because their native acquisition acknowledgement does not establish that safety contract. Capture checks the verified live connection and waits for acknowledgement of acquired native composition. Before every write the host rechecks exact owned text and caret; a 250 ms read-session probe covers missed notifications. Cancellation/platform termination restores selected original text only while the range still contains the host's own draft. User-modified bytes are retained. Final acknowledgement follows successful `EndComposition`.

UI status polls the helper on a background worker. Route sampling uses a live focused activation or a small native source query, without helper launches on the UI thread. Observation failures remain visible and conservatively choose owned delivery so capture can refuse explicitly.

Enablement requires both the per-user `IsEnabledLanguageProfile` result and the modern profile's enabled catalog flag. The development host had legacy enablement TRUE but catalog flags0 and no Arion menu entry: treating the legacy API alone as success was incorrect. The verified cause was registration with `bEnabledByDefault=FALSE`. A user-approved [`EnableLanguageProfileByDefault(TRUE)`](https://learn.microsoft.com/en-us/windows/win32/api/msctf/nf-msctf-itfinputprocessorprofiles-enablelanguageprofilebydefault) repair changed catalog flags to2; ordinary-user enable/select succeeded and fresh queries identified Arion active. Production registration now uses TRUE; it does not select Arion as the user's default keyboard.

The ordinary-user helper also calls the documented [`InstallLayoutOrTip`](https://learn.microsoft.com/en-us/windows/win32/tsf/installlayoutortip) with exactly Arion's profile and flags zero, preserving other TIPs and defaults. List membership alone did not prove visible installation. A successful UAC install is followed by bounded registration observation; real helper/query failures remain explicit. Selection requires an exact observed profile match; S_FALSE or a merely scheduled activation is not success. Actual user confirmation of Win+Space visibility remains separate from these native results.

Explicit Enable/Select first attempts ordinary-user enablement and queries the result. Only if the same validated registration still reports disabled does that action request UAC for one `repair`, then retry ordinary-user enablement and observe it before selection. A normally disabled user profile that enables successfully does not trigger UAC. Repair does not recopy the DLL, run on startup or repeat automatically in a loop; denial/failure stays visible and leaves retry available.

## Install, select and remove

1. Build/install a package containing `ime/ArionTSFLoader.dll`, `ime/ArionTSF.dll`, `ime/arion-ime.exe`, the schema-2 `ime/runtime.json` and the complete adjacent `keyboard` runtime. The shared Pinyin package still requires the acceptance gates below.
2. Choose Install in Arion Settings → Keyboard and accept Windows administrator approval. Cancellation/denial leaves retry available.
3. Enable/select Arion. The new profile belongs under **Chinese (Simplified, China)** in the Windows input selector. Verify its presence and active state in the target editor; Windows may remember a different source per application. API query success does not prove visible switcher acceptance.
4. Focus an editable field and use the dictation key. Actual composition acquisition is checked before microphone capture starts.

The stable CLSID is `{0CF9199C-BC90-44F0-9331-474FA6742271}`, profile `{78FBF1A9-3B48-4B37-A52C-B73D9D312263}`, current language `0x0804`, legacy language `0x0409`. The helper copies verified bytes to `%ProgramFiles%\Arion\InputMethod\<sha256>\ArionTSF.dll` and its adjacent `keyboard` directory, requires Administrator/SYSTEM ownership and rejects writable, unknown-ACE and reparse-point directories. COM/TSF registration is machine-wide; enablement is per user. Installation checks migration and build compatibility before copying any files. Disable/removal target only Arion's exact known language/CLSID/profile combinations.

The registered categories are keyboard and [`GUID_TFCAT_TIPCAP_SYSTRAYSUPPORT`](https://learn.microsoft.com/en-us/windows/win32/tsf/predefined-category-values), the documented tray capability for services without Windows Store app support. `IMMERSIVESUPPORT` is not declared. Category repair alone was insufficient on the development host; the subsequent default-enable repair resolved native catalog selection. The dedicated **repair** command validates the already registered owned DLL and fixes only its default enablement/categories, without copying a DLL or accepting an arbitrary path. The actual production helper's UAC repair and ordinary-user enable/select/query passed with the original DLL hash unchanged. Different builds use `update SOURCE NEW_SHA OLD_SHA`: complete new bytes are staged and verified in an independent protected version, then COM/profile registration switches. Normal failure restores prior registration metadata. A protected atomic intent marks interrupted transactions for completion, and query refuses corrupt receipts. Old loaded DLLs and their dependencies are retained; editors are not killed and user learning data is not removed. New editor processes use the new registration. The [Rime comparison](../research/rime-input-method-updates.md) records upstream sources, privilege boundaries and acceptance limits.

Manual diagnostics from the repository root follow. **Install**, **repair** and **remove** require Administrator PowerShell; run **enable**, **select**, **disable** and **query** as the intended desktop user. Existing broken registration can be repaired with the same payload. Before machine removal, run **disable** as that user to remove only Arion's TIP from their enabled list (`ILOT_UNINSTALL`, `0x1`). Settings performs both stages; declining the subsequent UAC prompt leaves machine registration available for enablement or removal retry.

```powershell
$arionIMERoot = (Resolve-Path apps/desktop/resources/windows-amd64/ime).Path
$arionIMEManifest = Get-Content "$arionIMERoot/runtime.json" -Raw | ConvertFrom-Json
& "$arionIMERoot/arion-ime.exe" install "$arionIMERoot/ArionTSF.dll" $arionIMEManifest.files.'ArionTSF.dll'
# Administrator, only when repairing an existing owned registration:
& "$arionIMERoot/arion-ime.exe" repair
& "$arionIMERoot/arion-ime.exe" enable
& "$arionIMERoot/arion-ime.exe" select
& "$arionIMERoot/arion-ime.exe" query
& "$arionIMERoot/arion-ime.exe" disable
& "$arionIMERoot/arion-ime.exe" remove
```

The fixed COM entry point is `%ProgramFiles%\Arion\InputMethod\ArionTSFLoader.dll`; the registry's `InprocServer32` default names it and its `CorePath` value names the protected `<sha256>\ArionTSF.dll`. The helper verifies the loader's compiled hash and the core/keyboard closure. Core-only upgrades atomically change `CorePath`, preserving the profile, categories, user enablement and current composition. A stable `ITfTextInputProcessorEx` facade releases its core delegate on deactivation and resolves the current core on the next activation, even when Windows caches the facade or factory. Outstanding core objects, server locks and in-flight calls prevent unloading; an active compatible core continues until safe release. Layout and transport callbacks hold core lifetime leases through their member cleanup.

The loader is deliberately independent of keyboard manifests and built with fixed PE image base and no timestamp, with an identical-source/different-output-directory hash gate. Its ABI and bytes are frozen for ordinary core updates. Changing the loader itself requires separate controlled maintenance; replacing a loaded bootstrap or accepting an arbitrary older loader is not implemented.

This architecture cannot alter a direct legacy DLL already loaded before the loader existed. That one-time migration needs Windows to release the old component; changing registration alone cannot add Pinyin or private-scope safety to its existing objects. New loader activations use the repaired core. Voice admission accepts only protocol 2 and validates the uniquely claimed loaded core even if COM retains another legacy module mapping in the process. Explicit removal reserves the loader and validated current closure before unregistering; a loaded file refuses removal without disturbing registration. The updater never kills editors, removes other input methods or recursively deletes Program Files.

## Build and tests

Hosted Windows CI runs the Go desktop and native keyboard suites under
the disposable standard-user launcher. The administrator runner token and its
redirected temporary directory do not satisfy the pipe-identity and immutable
path test preconditions. These checks remain mandatory; production identity and
path validation are not relaxed for CI.
Actual OS-key native window tests run in the runner's original interactive user
session after the Windows platform tests. The secondary standard-user account
can create windows and ACP contexts but cannot reliably acquire foreground/input
ownership over the administrator desktop. These two test scopes remain distinct.

`node scripts/build-windows-input-method.mjs` stages the pinned Rime sources, shared schema/data, native keyboard module and TSF binaries. It requires x64 MinGW-w64 GCC; `CC`/`CXX` select that toolchain, and the engine and private encrypted-database adapter must use the same C++ ABI. First provisioning can download hash-pinned public dependencies; installed typing requires no network or external Rime installation. Package verification checks x64 imports, exact runtime closure and hashes, then invokes the packaged helper's read-only `verify <sourceDLL> <expectedSHA>` operation to check its compiled keyboard-manifest binding. This operation does not register a profile or request elevation.

`pnpm keyboard:windows:check` builds the runtime and runs shared core/storage, real conversion, dynamic module/filesystem and owned ACP editor tests. The ACP fixture manually calls key/focus callbacks into real TSF text stores; it does not prove registered system event dispatch. Native menu/mode/voice acceptance, real installed editors and a fresh macOS regression remain required before claiming the shared implementation complete.

The installed protocol-2 build has separately passed an owned WebView2 system test with actual OS keys, without a focus-away/back workaround: textarea and contenteditable both committed `你好`; the production TSF adapter verified the real pipe peer and delivered a synthetic voice draft/final commit; a DOM-confirmed password field refused composition, and ordinary-field capture/cancel recovered without changing committed text. The test restored the previous profile and foreground and exited cleanly. Its local source/binary/adapter hash ledger and native DOM event log are retained in `.workspace/windows-tsf/current-defects/webview-installed-protocol2-final.json`. This verifies delivery and field safety, not microphone recognition or interaction with Codex/user documents; representative third-party editor and macOS acceptance remain separate.

That earlier composition test did not check candidate visibility. A later clean reproduction found 16 of 20 composing samples hidden in the previous `0a9b3af4...` core. After the layout repair, the installed `a216e37b...` core and fixed loader passed the full ordinary/rich-field matrix: all 20 per-letter samples were visible at 160 ms and after another 500 ms; both fields committed Chinese; the production voice adapter committed synthetic Unicode, refused the password field and recovered cancellation. No refocus or layout-hook activation was used. The clean report is `.workspace/windows-tsf/current-defects/webview-candidate-layout-installed-final.json`; it records the actual core/loader/keyboard/Rime hashes and successful profile restoration. An earlier interrupted run is retained separately as failed. This is not proof of interaction with an already open ChatGPT process or real-core cross-version handover in that application.

From `apps/desktop`, outside a low-integrity sandbox:

```powershell
go test ./infrastructure/imebridge ./infrastructure/tsf ./bootstrap
go test -tags arion_e2e ./infrastructure/imebridge -run TestWindowsNativeTSFVoice -count=1
```

The native test requires `.workspace/windows-tsf/arion-tsf-fixture.exe`, compiled from `ime/windows/fixture.cpp` with C++17 and `ole32`, `oleaut32`, `uuid`, `advapi32`. It uses documented RichEdit `SES_USECTF`. `ARION_TSF_FIXTURE` selects the independent `textstore_fixture.cpp` executable; `ARION_TSF_FULL_STORE=1` additionally tests owner refusal and actual deferred locks. Both use synthetic owned documents without machine registration or existing-document changes. `transport_test.cpp` separately verifies peer/integrity checks, stalled-server deadlines and I/O shutdown. Private logs are under `.workspace/windows-tsf/`.

Native primitive success is distinct from complete setup acceptance. The development host has observed UAC registration and per-user enablement; complete application install→enable→select, UAC removal, per-app remembered-source behavior, installed-package speech and a representative editor matrix remain acceptance gates until observed.
