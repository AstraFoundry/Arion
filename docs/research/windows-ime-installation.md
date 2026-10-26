# Windows IME installation: Doubao and Chatterfly

## Scope and evidence

This investigation examines the two user-supplied Windows installers to compare
their TSF registration and deployment with Arion's missing input-switcher entry.
Neither installer nor any extracted application/DLL was executed or registered.
Static registration paths do not prove installation success or editor behavior.

The requested [REA skill](https://github.com/morluto/rea/blob/main/.agents/skills/reverse-engineer-anything/SKILL.md)
was followed using the published REA 6.7.0 CLI and existing Node 24.22.1.
The published CLI has fewer tools than repository-main skill version 46.
No native decompiler engine was configured. REA artifact/resource inspection and
its GNU objdump adapter were used; additional extraction, bounded disassembly
and PE/GUID decoding are explicitly ordinary-tool evidence, not REA pseudocode.
No Codex configuration was changed.

| Installer | Bytes | SHA-256 |
| --- | ---: | --- |
| DoubaoIME_Installer_0.9.2.7_release.exe | 123961912 | `d777027cc822a67dac04ba502616f6e6216a54c06681e21f8f45ef0a89cec5db` |
| Chatterfly_V1.0.0.5764.exe | 164586224 | `a97e1e4ce16bb71a47443e95b4280208f244feec33679e9c1bac557fefbd1c1c` |

Windows Authenticode inspection reported valid signatures for both supplied
files. Doubao's container is Inno Setup 6.7.0; Chatterfly is NSIS-Park-3 Unicode.
Existing 7-Zip 25.01 extracted Chatterfly. Official innoextract 1.9 could not
parse Doubao; the maintained `UserUnknownFactor/innoextract_win` release 670
extracted it successfully. Its downloaded ZIP SHA-256 matched the release asset
digest `79b69b9b1fcd98f42ccd4b245efdf6a03bcfb674ba6af482f5a46891c9ed4d14`.

Complete local evidence is under `.workspace/rea-windows-ime/`, including
installer inventories, extracted-file identities, original REA JSON and bounded
assembly. Proprietary binaries are not committed to this repository.

## Registration comparison

| Property | Doubao 0.9.2.7 | Chatterfly 1.0.0.5764 | Current Arion |
| --- | --- | --- | --- |
| Primary registration language | Simplified Chinese `0x0804` | Simplified Chinese `0x0804` | English US `0x0409` |
| `RegisterProfile` default enabled | TRUE | TRUE | FALSE; separate per-user enablement |
| `RegisterProfile` flags | 0 | 0 | 0 |
| Substitute keyboard layout | Derived Chinese layout handle, with fallback path | NULL | NULL |
| Packaged IME architectures | x64 and x86 | x64 and x86 | x64 only |
| Registered capability categories | 16 GUIDs | 9 GUIDs | Keyboard and system tray |
| User-list installation | Further helper-path analysis needed | `InstallLayoutOrTip`, flags 0, `0804:%s%s` format | Exact owned TIP, flags 0 |

Doubao's `tsf-oime-core.dll` has SHA-256
`ef827968a374de491529f65eade36d58c7eab78933ebe8508bd2fed8563290c0`.
Its `DllRegisterServer` is at RVA `0x36100`. The profile-manager IID at
RVA `0x185468` and vtable slot `0x40` identify the registration call at
RVA `0x367d1`; argument dataflow supplies `0x0804`, TRUE and flags zero.
The service CLSID at RVA `0x174ad0` is
`{9D2B2E2B-3C93-4D2F-9D35-6EEB85F0D2B0}`; the profile at `0x17d400` is
`{2B4D4B3A-4D4F-4C0A-8E66-7F771A2B9C10}`.

Chatterfly's `ChatterflyTSF64.ime` has SHA-256
`abda434776540f5fe8b8e2c7ea82c848c57e875ee57a4fa008a5df7865cecc55`.
Its `DllRegisterServer` is at RVA `0xa50e0`; the registration call at
`0xa521c` supplies `0x0804`, TRUE, NULL substitute layout and flags zero.
The CLSID at `0x4321f8` is `{604A99E3-6D90-4571-824D-2639BD572F6C}`;
the profile at `0x432248` is `{C16C250B-CF1C-4C58-B068-12AEDBED610F}`.
Static code registers the profile, then categories, then installs the user TIP.

Both declare immersive, system-tray, UI-element and COM-less support. These are
capability declarations, not harmless visibility toggles. They cannot be copied
into Arion before the corresponding runtime behavior is implemented and tested.
The presence of both x86/x64 payloads establishes packaging, not verification of
every host architecture or an explanation for the x64 switcher's omission.

## Deployment structure

Doubao has a stable `bootstrap/` layer with `tsf-oime.dll`, its x86 counterpart,
`ImeWatchdog.exe`, `SettingsLauncher.exe`, `UpdateTaskHost.exe` and
`oime-installer.exe`, enumerated by `bootstrap_manifest.json`. Versioned business
code lives under `versions/v0.9.2.7/`, including `tsf-oime-core.dll` and separate
settings, UI and RPC components.

The loader's exported `DllRegisterServer` at RVA `0x1cd0` invokes its loader
routine and an indirect registration target. The routine resolves named exports
with `GetProcAddress`; the registration target is stored at RVA `0xbac00`.
This establishes an actual native indirection layer rather than inferring it
only from the directory names. Its update ordering and recovery guarantees
have not been exercised.

The extracted Doubao tree contains 195 files, 315772961 bytes and 45 PE files.
The installer helper includes separate `--install-tsf` machine/elevated and
`--install-tsf-user` user-only diagnostics, plus an elevated
`--commit-version-dir` operation (file offsets `428848`, `429328`, `429392`,
`429472` and `429496`). These are explicit shipped operation declarations;
their full conditional execution paths have not been reconstructed. A packaged
update-task template declares a boot trigger and SYSTEM principal.
`DoubaoImeSettings.exe.config` declares .NET Framework 4.6; REA managed metadata
for `DoubaoIme.Settings.UI.dll` references WindowsBase, PresentationFramework,
PresentationCore, System.Xaml and WPF-specific SharpVectors assemblies. This
establishes a separate WPF settings component alongside the native TSF.

Chatterfly includes distinct `ChatterflyPY.ime` and `ChatterflyTSF.ime` variants,
their 64-bit counterparts, `ChatterflyImeBroker.exe`, `ChatterflyWin10Setup.exe`,
`ChatterflyComMgr.exe` and `Install64.exe`. This is a multi-component installation,
not merely an application executable with a floating window. Exact broker
responsibilities and the complete installer execution order remain outside the
bounded registration analysis.
The inspected Chatterfly registration routine does not check the BOOL result of
`InstallLayoutOrTip` before its success return, another reason that a successful
registration entry point cannot establish actual user-menu visibility.

## Implications for Arion

The strongest controlled follow-up is to test language placement and default
enablement independently, with original profile state recorded and restored.
Both competitors and Microsoft's sample default-enable their registered profile;
Arion currently differs. This is a hypothesis for testing, not a proven fix.
Moving Arion to Chinese also requires preserving its multilingual voice contract
and migrating only its own profile; it must not overwrite Microsoft Pinyin.

Arion's actual registered DLL can be created through COM and initialized with
`ActivateEx`. Its failing gate is system catalog visibility/selection. The
observed user switcher still contains only Pinyin, Japanese IME and US keyboard.
Registry presence, per-user enablement and a successful activation HRESULT are
insufficient acceptance. The next implementation must require visible selection
and real editor composition/commit, followed by restart and update persistence.

The stable-loader/versioned-core split is a useful separate lifecycle design to
evaluate. It does not itself resolve the current missing switcher entry, and
loaded old versions still require an explicit handover/cleanup policy.

## REA evidence references and remaining limits

### Loader lifecycle follow-up

The reviewed Doubao bootstrap has direct load/unload control flow, beyond an
import-table hint. Loader RVA `0x1240` returns its cached core when present;
otherwise it reads `HKLM\SOFTWARE\DoubaoIme\VersionDir`, appends
`\tsf-oime-core.dll`, and calls `LoadLibraryExW` at `0x1370`.
`DllGetClassObject` calls this loader at `0x1c68` and forwards to the cached
factory. `DllCanUnloadNow` calls the core's unload check and only on `S_OK`
calls `FreeLibrary` at `0x1bfa`, then clears the cached module and exports.
The next factory request can therefore read the current version directory.
Local disassembly is retained in
`.workspace/rea-windows-ime/doubao-tsf-rea-disassembly.asm`.

This supports a stable bootstrap with reloadable versioned core. It does not
prove proactive unload timing, transfer of an active composition, or seamless
updates in every editor. Restarting the entire editor is not an unconditional
TSF requirement. Arion's earlier directly registered versioned DLL still has
its own lifetime: accepting a compatible older voice protocol cannot add
Pinyin support to an already loaded voice-only implementation.

- Installer artifact: Doubao `ev_5d22f537122fb09b9178ff5af1d534d06bd22aeb6f30edc6d6e6307d77179a0c`.
- Doubao loader disassembly: `ev_17531b9f410f40f61f69a5dcef80a2ab549a83dc12d11c606e593f8fb4eee429`; completed, untruncated, unchanged artifact.
- Doubao core headers: `ev_56b9954f58fcdcfbba51bb2899439037c47bf16361c3e8bc62dc291a5ef6a04a`.
- Chatterfly TSF64 headers: `ev_3e2110e8d0e98368b9084a71822ee2893a061dc5ff6020936174772e15120720`.
- Doubao settings managed metadata: `ev_fa94fc7cb965b83fb91226ab41f05fca61e81ab855620b7ca69e90e2e7becc4c`.

REA's installer graph recognized the root PE only; it did not decode the Inno
or NSIS payload. Separate extractor inventories supply that evidence. Detailed
registration argument/GUID findings use ordinary bounded objdump/PE decoding.
Installation, onboarding, speech accuracy, latency, AppContainer compatibility,
network behavior and update rollback have not been observed in either app.

Primary API references: [Microsoft IME requirements](https://learn.microsoft.com/en-us/windows/apps/develop/input/input-method-editor-requirements),
[RegisterProfile](https://learn.microsoft.com/en-us/windows/win32/api/msctf/nf-msctf-itfinputprocessorprofilemgr-registerprofile),
[Microsoft SampleIME registration](https://github.com/microsoft/Windows-classic-samples/blob/main/Samples/IME/cpp/SampleIME/Register.cpp).
