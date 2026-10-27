# Windows TSF native host

`service.cpp` runs inside the editor's COM apartment. The message window
dispatches pipe messages back to that apartment; only TSF edit sessions touch
context, selection and composition objects.

`loader.cpp` is the stable COM entry point; `service.cpp` remains the protected
hash-versioned core. The loader's processor facade resolves the machine-owned
`CorePath` when activated and releases the delegate on deactivation. Cached
outer COM objects can therefore use a new core after a normal source lifecycle,
while live core objects, locks and calls prevent unsafe unload. Core publication
does not toggle the user's profile. Direct pre-loader components still require
their original native lifetime to end before this mechanism can apply.

`node scripts/test-windows-tsf-loader.mjs` exercises real owned fake-core DLLs,
cached factories/facades, active references, server locks, failed loading,
production path refusal, deterministic bootstrap bytes and callback rollback.
It never registers a system source or changes an editor.

`transport.h` keeps pipe I/O off the editor thread. Its outbound queue is
bounded to 64 messages and 1 MiB, with a one-second deadline for each message.
Read and write workers use separate overlapped operations and a shared stop
event. Disconnect cancels pending I/O and joins workers before releasing the
pipe, event handles or queued messages. It does not cast a C++ thread handle
into a Windows thread handle.

The native host supports ordinary desktop processes only: low-integrity,
high-integrity and elevated hosts are refused. The pipe server must have the
same user SID, logon session and integrity level as the host, and must also be
unelevated. The per-user SID pipe ACL and first-instance listener reservation
remain independent protections. Build hashes identify matching payload bytes;
they do not authenticate arbitrary processes running as the same user.

## Shared keyboard adapter

`KeyboardClient` verifies the complete sibling `keyboard/runtime.json` closure
against the hash compiled into the service before loading `ArionKeyboard.dll`
with an absolute path and restricted dependency search. The uniquely named
`ArionRime.dll` avoids a dependency-name collision with another vendor's Rime.
The COM service itself imports no Rime DLL; voice ownership remains separate
from optional keyboard availability.

`typing.h` calls the shared C session, key-policy, Shift-tap and candidate-menu
APIs only after TSF grants a write edit session and the selected field passes
the host's safety checks. Test-key callbacks never process an engine key.
Candidate commits use the engine result, not displayed candidate strings.
Home/End and UTF-16 caret positions, prefix selection, suffix continuation and
rollback stay in the shared engine. The native surface consumes its ranked
engine indices, including the six-row expanded menu, and publishes a TSF
candidate UI element with that exact ordering.

Candidate data preparation runs only in the write edit session. Geometry is
also refreshed after `ITfTextLayoutSink::OnLayoutChange` through an asynchronous
`TF_ES_READ` session. Browser layout may return `TS_E_NOLAYOUT` immediately
after `SetText`; the native surface stays hidden until the current caret has
a fresh, unclipped rectangle. No previous rectangle is reused. Queued reads
are coalesced and bound to the current context, activation and composition
revision; cancellation, focus loss and user edits cannot revive old candidates.
The read path never processes an engine key, changes engine selection, writes
text or starts a composition. See the upstream
[Weasel layout sink](https://github.com/rime/weasel/blob/82b50c9e9b9ee0bd31357db6f25ddc804546c7f7/WeaselTSF/TextEditSink.cpp#L59)
and its [read edit session](https://github.com/rime/weasel/blob/82b50c9e9b9ee0bd31357db6f25ddc804546c7f7/WeaselTSF/Composition.cpp#L217).

Width and punctuation choices live in
`%APPDATA%/Arion/keyboard/options.json`. A field-specific, locked read/modify/
atomic-replace operation preserves another editor's independent choice. A
failed save changes neither the engine mode nor an existing draft; changing a
mode without a draft does not acquire a composition or collapse selection.
The app-owned `%APPDATA%/Arion/keyboard.json` controls page size, candidate
size, appearance, resolved accent, context ranking, Emoji suggestions and the Voice Input key
title. The host refreshes preferences independently of the desktop app.

The 23-DIP mode badge is nonactivating and click-through. Its opaque rounded
surface appears immediately for one second; only its old/new label flips in
two 120-ms halves, with client-area animations disabled when Windows requests
reduced animation. Missing caret geometry suppresses it. Typing, voice
takeover and focus loss dismiss it.

The optional module serializes its engine API and shares the encrypted
learning/ranking implementation with macOS. Windows learning cooperates across
editor processes using short locked, authenticated reload-and-merge writes;
it does not remove locking or create per-editor plaintext databases. Context
learning is posted outside the key/edit callback, and only a successful client
commit schedules it. Reset finalizes local engine/context databases before
deleting the vault and advances the generation so every local client drops
its expired session. A peer-held vault refuses reset without deletion and
recovers the unchanged local runtime; authentication failures remain closed.

## Owned keyboard validation

`node scripts/test-windows-tsf-keyboard.mjs` compiles the production module and
an explicitly marked test TSF service into a complete isolated bundle. It uses
a private synthetic user profile and unique authenticated test pipe; it never
changes the installed registration or the real app's pipe. It records source
and runtime/import-library hashes and rejects changes during the run.

The ACP editor supplies actual UTF-16 text, selection, composition acceptance,
read-only/input-scope properties and lock grants. Its manual client cannot
install a system foreground TIP key sink or enumerate the reserved system
input-mode menu as an active registered TIP. The test-only build therefore
calls the real COM key/focus/menu objects explicitly. This is a layered test
of actual TSF edits, not evidence of installed system key dispatch, taskbar
menu visibility or remembered per-app input selection.

The expanded matrix covers Chinese/raw commits, selected Unicode cancellation,
password/read-only/owner refusal, no processing before a granted lock, external
user edits, focus cleanup, zero-width caret geometry, accessible candidate
ordering, number/mouse/grid selection, partial-prefix rollback, caret editing,
Shift/chords/duplicate and anonymous modifier notifications, persisted modes
and save refusal, badge geometry/lifecycle, reset, configuration refresh, and
authenticated voice/keyboard takeover with stale voice results and disconnect.
Delayed-layout cases return `TS_E_NOLAYOUT` after every letter of `nihao`,
then permit only read locks and issue actual text-store layout notifications.
They check candidate recovery without document writes, fresh positioning after
scrolling, clipping suppression and refusal to revive a focus-revoked draft.
This is separate from installed WebView2 acceptance, which must observe native
candidate visibility after each key in both textarea and contenteditable;
successful final Chinese commits alone do not pass that acceptance.

The installed layout-fix build was exercised through actual Windows key
dispatch in owned WebView2 textarea and contenteditable fields. All 20 native
candidate samples were visible across `n`, `ni`, `nih`, `niha`, `nihao` at
160 ms and another 500 ms, with no refocus or layout-hook workaround. Both fields
committed Chinese. The same run verified production-adapter voice commit,
private/password refusal and subsequent ordinary-field capture/cancel, then
restored the previous input profile and exited. This establishes those owned
browser cases, not every application's native integration.
`native_keys_test.cpp` and `candidate_window_test.cpp` also run through this
entry. The commit-failure cleanup/learning branch has a source-level guard;
this matrix does not inject an `EndComposition` failure.

## Build entry point

`node scripts/build-windows-input-method.mjs` is the single build pipeline.
It stages the pinned Rime SDK and data, compiles `ArionKeyboard.dll` with
`RIME_IMPORTS`, records its hash, then binds the final keyboard manifest into
the TSF DLL and registration helper. The compiler is selected with `CC`/`CXX`
in the environment; the private adapter and Rime use the same MinGW-w64 GCC
ABI. Outputs remain `resources/windows-amd64/ime` and the sibling
`resources/windows-amd64/keyboard` closure.

`CMakeLists.txt` only delegates its default `ArionTSF` target to that Node
entry. It has no second compiler configuration, SDK selection or output
layout. Legacy `ARION_OUTPUT`, `ARION_KEYBOARD_SDK` and
`ARION_KEYBOARD_MANIFEST_SHA256` cache parameters are rejected, including empty
values. Use a fresh CMake build directory when migrating from the old entry.
The manifest binding is always computed after the keyboard module is built.

## Language bar contracts

`language_bar.h` exposes the system `GUID_LBI_INPUTMODE` item required by
[Windows 8 and later](https://learn.microsoft.com/en-us/windows/win32/api/ctfutb/nf-ctfutb-itflangbaritem-getinfo).
Its button and menu styles provide a mode-toggle click and a native
[`ITfMenu` entry point](https://learn.microsoft.com/en-us/windows/win32/api/ctfutb/nf-ctfutb-itflangbaritembutton-initmenu).
Native TSF and right-click menus share their ordering, localized labels and
disabled states. Dispatch checks current availability again, including voice
connectivity and pending-text recovery. Reset and Open Arion remain available
when typing is unavailable. The item supports the hidden-status control style;
`Show` publishes status changes and mode changes invalidate icon, text, tooltip
and status. Icons use system small-icon dimensions and transfer ownership to
the caller, which must call `DestroyIcon`.

`node scripts/test-windows-language-bar.mjs` compiles strict native COM
contracts and records compiler, source and executable hashes under
`.workspace/tests/windows-language-bar`. It checks the input-mode GUID/styles,
COM identity and sink lifetime, native menu ordering and disabled rows, stale
command rejection, insertion failures, current labels, visibility notifications
and real GDI icon ownership. It uses mock menu/sink objects, performs no
registration or TSF activation, and does not establish that the installed
Windows taskbar displays or invokes the menu. That remains a separate
installed-source acceptance check.

## Registration observation

The current Pinyin profile is **Chinese (Simplified, China) → Arion**,
language `0x0804`. Its CLSID and profile GUID remain unchanged. The old
English `0x0409` Arion Voice profile is a legacy installation migrated by the
same automatic update transaction as a different installed build. A complete
new hash-versioned closure is published before registration switches; old
DLLs and dependencies remain intact for editors that still hold them.

The helper enumerates all languages (`EnumProfiles(0)`) and matches the exact
type, language, CLSID and profile GUID. `query` retains the actual old build
and returns `registered:true`, `legacy:true`, `upgradeRequired:true` for an
existing voice profile. Dangling owned COM/profile registrations also remain
visible as requiring recovery. Registry/enumeration errors are failures,
not evidence of absence. Fresh `install` checks migration/build compatibility
before creating directories or copying files. `update SOURCE NEW_SHA OLD_SHA`
compares the expected previous registration, stages complete protected bytes,
rechecks selection, switches COM/current-profile registration and retires only
the owned legacy profile. It restores prior registration/category metadata on
failure and retains an atomic old/new-hash intent until commit or successful
rollback; query marks interrupted registration as requiring completion.
Ordinary-user legacy-list cleanup/enablement follows machine publication.
Enable, select and repair still reject an unmigrated legacy or mixed catalog.
Disable/removal target only the
observed known language/profile identities; unregister uses flags zero rather
than `TF_URP_ALLPROFILES`, and cleanup retains the existing payload-lock guards.

`verify SOURCE_DLL EXPECTED_SHA256` reads the source DLL and adjacent keyboard
closure against the manifest hash compiled into the helper. It runs before
COM initialization, creates no files and changes no registration. Release
package verification applies it to the actual package payload.

`node scripts/test-windows-ime-registration.mjs` compiles the callback-only
registration, category, selection, user-TIP and migration contracts, plus the
independent read-only query regression. Its verifier cases use clearly
synthetic file bytes to check a matching closure, wrong source hash and wrong
compiled manifest binding without loading a DLL. Reports include exact
commands/source hashes under `.workspace/tests/windows-ime-registration`.
With no existing source the query records exit 77; `--require-existing`
requires that independent native observation to pass. Neither mode installs,
enables, selects or removes an input source, and neither invokes UAC.

The default-enable/category observations below describe the earlier voice
host. They are separate from the versioned migration transaction described
above. See the [pinned Rime comparison](../../../../docs/research/rime-input-method-updates.md)
for automatic-update policy, UAC and retained-version limits.

The helper requires both `ITfInputProcessorProfiles::IsEnabledLanguageProfile`
and `ITfInputProcessorProfileMgr`'s enabled catalog flag. The development host
exposed a real broken state: the legacy API returned TRUE while modern flags
were zero, the source was omitted from Win+Space, and catalog activation did
not select Arion. The earlier interpretation that the legacy API alone proved
enablement was incorrect.

The verified cause was machine profile registration with
`bEnabledByDefault=FALSE`. A coordinated user-approved call to
`EnableLanguageProfileByDefault(TRUE)` changed the modern flag from zero to2;
ordinary-user enable/select then succeeded, and independent queries reported
the exact Arion profile active (flags3). The DLL and language0409 were unchanged.
Production `RegisterProfile` now uses TRUE. The dedicated administrator
`repair` command validates the existing owned DLL, repairs default enablement
with that official API and the two managed categories, and neither copies a
DLL nor accepts a replacement path. It does not change the user's default
keyboard or other input methods. The production helper's actual UAC repair
and subsequent ordinary-user enable/select/query also passed. Actual user
confirmation of the Win+Space menu remains a separate pending check.

The desktop's explicit Activate flow enables as the ordinary user and queries
the result first. Only a still-disabled, matching validated registration
triggers one elevated `repair`, followed by ordinary-user enablement and
observation before selection. Normal per-user disablement does not require
UAC if enablement succeeds. This repair neither recopies the DLL nor runs on
startup or as a repeated automatic registration attempt.

`select` keeps its session request flags but treats S_FALSE as failure and
requires an exact observed type/language/CLSID/profile match before success.
A successful scheduled request for a different active language is insufficient.
After a successful elevated installation, the desktop observes registration
for up to 30 seconds rather than requiring immediate cross-process discovery.
Failed helper operations and failed native queries still stop immediately.
Activation errors identify discovery, enablement or selection; an enabled
profile is distinct from switching the current input language to its profile.

`enable` calls system32 `input.dll!InstallLayoutOrTip` for exactly Arion's
`0x0409:{CLSID}{PROFILE}` with flags zero. Legacy enablement alone was true on
the development host while the Win+Space list omitted Arion. The documented
per-user install step added its English (United States) entry without changing
the existing Chinese/Japanese TIPs. This is distinct from actual selection or
visual verification of Win+Space. `disable` uses only `ILOT_UNINSTALL` (`0x1`)
for the same TIP; Settings runs it as the desktop user before elevated machine
removal. Neither command replaces the language list, sets a default, or uses
`ILOT_CLEANINSTALL`. See Microsoft's [installation requirements](https://learn.microsoft.com/en-us/windows/apps/develop/input/input-method-editor-requirements#ime-installation)
and [InstallLayoutOrTip contract](https://learn.microsoft.com/en-us/windows/win32/tsf/installlayoutortip).

The helper also registers `GUID_TFCAT_TIPCAP_SYSTRAYSUPPORT` alongside the
keyboard category. Microsoft [documents this capability](https://learn.microsoft.com/en-us/windows/win32/tsf/predefined-category-values)
for services that support tray inclusion without declaring Windows Store app
support. It does not register `IMMERSIVESUPPORT`. Read-only host diagnostics
found only the keyboard category and `dwCaps=0` even after exact Arion TIP
membership appeared in `Get-WinUserLanguageList` and native enumeration;
Windows Settings still omitted English. These APIs therefore do not establish
visible switcher/settings acceptance. Installing the same validated DLL now
repairs these two managed categories without replacing the DLL or registration
path. A different DLL remains refused. Removal unregisters both categories.
Actual Win+Space visibility and selection must be checked after the repair.
On the development host, the user-approved same-DLL repair exited zero;
independent post-repair COM queries observed both exact categories and
`dwCaps=0x00020000` (system-tray support), with the original DLL hash unchanged.
At that point the source had legacy enablement but no catalog enablement and
was not selected. These results established the
registered capability, not visual switcher/settings acceptance.

`category_test.cpp` verifies exact managed categories, symmetric cleanup,
same-DLL repair, different-DLL refusal and native failure propagation using
callbacks, without invoking OS registration:

```powershell
g++ -std=c++17 -DINITGUID -O2 -Wall -Wextra -Werror -static -municode apps/desktop/ime/windows/category_test.cpp -o .workspace/windows-tsf/category-test.exe -lole32 -loleaut32 -luuid -ladvapi32 -lshell32
& .workspace/windows-tsf/category-test.exe
```

The callback contract regression verifies exact enable/remove TIP strings,
zero/one flags and native failure propagation without changing OS settings:

```powershell
g++ -std=c++17 -DINITGUID -O2 -Wall -Wextra -Werror -static -municode apps/desktop/ime/windows/user_install_test.cpp -o .workspace/windows-tsf/user-install-test.exe -lole32 -loleaut32 -luuid -ladvapi32 -lshell32
& .workspace/windows-tsf/user-install-test.exe
```

`control_test.cpp` is a read-only regression against an already registered
Arion source. It compares production query output with both enablement APIs
without changing registration, enablement or selection:

```powershell
g++ -std=c++17 -DINITGUID -O2 -Wall -Wextra -Werror -static -municode apps/desktop/ime/windows/control_test.cpp -o .workspace/windows-tsf/control-test.exe -lole32 -loleaut32 -luuid -ladvapi32 -lshell32
& .workspace/windows-tsf/control-test.exe
```

The regression passed against the development-host profile with both user and
catalog enablement true. `registration_test.cpp` guards TRUE as the exact owned
profile's registration/repair default and checks both enabled-state inputs;
`selection_test.cpp` rejects S_FALSE, native errors and mismatched profiles.
Both use callback contracts and perform no registration or selection:

```powershell
g++ -std=c++17 -DINITGUID -O2 -Wall -Wextra -Werror -static -municode apps/desktop/ime/windows/registration_test.cpp -o .workspace/windows-tsf/registration-test.exe -lole32 -loleaut32 -luuid -ladvapi32 -lshell32
& .workspace/windows-tsf/registration-test.exe
g++ -std=c++17 -DINITGUID -O2 -Wall -Wextra -Werror -static -municode apps/desktop/ime/windows/selection_test.cpp -o .workspace/windows-tsf/selection-test.exe -lole32 -loleaut32 -luuid -ladvapi32 -lshell32
& .workspace/windows-tsf/selection-test.exe
```

Without a registered source, `control_test.cpp` exits 77 rather than creating one or reporting
a passing source test. Go tests in `infrastructure/tsf` independently cover
delayed discovery, native-query failure, build mismatch and bounded discovery.

## Transport regression

From the repository root on Windows x64 with the pinned native compiler:

```powershell
New-Item -ItemType Directory -Force .workspace/windows-tsf | Out-Null
g++ -std=c++17 -O2 -Wall -Wextra -Werror -static apps/desktop/ime/windows/transport_test.cpp -o .workspace/windows-tsf/transport-test.exe -ladvapi32
& .workspace/windows-tsf/transport-test.exe
```

Run the executable from an ordinary medium-integrity user session. A
low-integrity sandbox or elevated shell exits with code 77 and prints exact
token diagnostics instead of bypassing the production integrity check.

The regression passed on the Windows x64 development host. It verifies peer
identity rejection, rejection of a duplicated low-integrity token, acceptance
of the same-user/session/integrity server, prompt queued sends, the one-second
timeout against an owned server that never reads, and cancellation/join of
pending reads and writes within 500 ms. It creates only owned named pipes and
a message-only window; it neither registers an input source nor changes the
user's primary token. The native editor fixture separately verifies actual
TSF composition and commit behavior.

## Native owned-editor matrix

`textstore_fixture.cpp` implements a visible native editor with a real
`ITextStoreACP` document, UTF-16 selection/ranges, read/write lock grants,
input-scope attributes and context-owner composition callbacks. External
fixture edits send the documented TSF change notifications; TSF-originated
`SetText` calls update the actual document without sending recursive change
notifications. It exposes counters for actual writes, accepted/refused
compositions and deferred locks. It contains no expected recognition results.

Build from the repository root, then run from an ordinary medium-integrity
Windows desktop session with the production `ArionTSF.dll` already staged:

```powershell
g++ -std=c++17 -DINITGUID -O2 -Wall -Wextra -Werror -static -municode apps/desktop/ime/windows/textstore_fixture.cpp -o .workspace/windows-tsf/arion-tsf-textstore-fixture.exe -lole32 -loleaut32 -luuid -ladvapi32
$env:ARION_TSF_FIXTURE = (Resolve-Path .workspace/windows-tsf/arion-tsf-textstore-fixture.exe).Path
$env:ARION_TSF_FULL_STORE = '1'
Push-Location apps/desktop/infrastructure/imebridge
go test -tags arion_e2e . -run '^TestWindowsNativeTSFVoice$' -count=1 -v
Pop-Location
```

The final ACP matrix passed in 1.19 seconds on the development host, with
native results retained locally in `.workspace/windows-tsf/acp-native-final.log`.
It verifies Unicode/combining-mark commit and duplicate rejection; restoration
of selected text on cancel, selection change and focus loss; password and
read-only acquisition refusal; native user replacement winning over late
updates/commit; context-owner refusal returning no capability; rejection of a
selection changed before a deferred lock grant; and cancellation while native
acquisition is deferred. The refusal and deferral cases observe actual store
counters before checking their outcomes.

The first ACP matrix failed focus-loss restoration and is not acceptance
evidence. The service repair was verified by the subsequent base and expanded
matrices. This fixture loads the DLL through its real COM class factory and
uses the real per-user pipe without machine registration, changing the selected
input source or accessing user documents. It does not establish first-install
UAC acceptance or compatibility with arbitrary third-party editors.

## Emoji candidates

The default-on `emojiSuggestions` preference enables the shared exact-word filter. Typed candidate metadata and caret refresh cross the validated keyboard DLL ABI; TSF selection commits the actual engine identity. Emoji-bearing commits, including a selected Chinese prefix plus an Emoji suffix, never enter Chinese context learning. Preference refresh rebuilds the active typed menu under the editor write lock and advances its revision.

`emoji_text.h` uses the installed Segoe UI Emoji font with DirectWrite shaping and Direct2D color drawing; no font/image resources are shipped. The filter's cached native check requires one complete cluster and one nonmissing color glyph. Unsupported flags, missing/newer glyphs and decomposed ZWJ sequences are omitted, preserving text candidate order. The candidate window uses Direct2D DC interop for typed Emoji rows while retaining GDI text and existing geometry/hit testing.

The owned native suite checks color pixels, light/dark surfaces, all three text sizes and focus retention, and captures actual candidate windows. Module and ACP tests cover metadata, exact selection, Space/punctuation, mixed commits, caret suppression/restoration, preference refresh and context-learning isolation. Installed Windows editor acceptance remains separate.
