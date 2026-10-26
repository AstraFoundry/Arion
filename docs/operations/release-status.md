# Desktop release status

This file accompanies each release as `RELEASE-STATUS.md`. The `0.1.0-alpha.15` release includes macOS Apple silicon and Windows x64 packages. Windows application/installer publisher signing is omitted for this preview; update archives still require Ed25519 signatures. Future macOS releases support Apple silicon only ([ADR 021](../decisions/021-apple-silicon-only-macos.md)); Intel macOS is unsupported. Build success verifies packaging; the evidence below limits the supported product claims.

| Capability | macOS Apple silicon | Windows x64 |
| --- | --- | --- |
| Native package | App and DMG | Per-user NSIS installer |
| Local speech recognition | Native MLX 1.7B 8-bit; 118 development fixtures measured | CPU/Vulkan and installed-product speech checks pass on one development host; mixed-language quality errors, publisher signing and clean-host acceptance remain open |
| Local polishing | Worker and pinned runtime bundled | Bundled CPU GGUF runtime; installed local-polishing checks pass on one development host |
| Optional cloud polishing | Explicit engine/provider/model choice; unit and view coverage; real-service acceptance open | Implemented with the same explicit choice; real-service/native acceptance open |
| System input method | Offline Pinyin/voice IMK host with horizontal candidates and context ranking; native 36/36 cases pass; installed physical Shift and dictation accepted locally in Codex/Cursor | Offline Pinyin and TSF voice composition; 40 native cases and 20 installed-editor candidate observations pass; broad editor acceptance remains open |
| Encrypted sync | Desktop enrollment and outbox pending | Desktop enrollment and outbox pending |

The table describes the alpha.15 source and packaging capabilities. Alpha.15 preserves expanded candidates during composition and renders the shared onboarding laptop as a CPU-rasterized bitmap to avoid missing keycaps and text during animation. Windows local GGUF polishing ([ADR 025](../decisions/025-windows-local-polishing.md)), TSF setup and updates ([runbook](windows-input-method.md)), and overlay lifecycle fixes are included. Early direct-loaded input-method DLLs already resident in editors need a native component reload; fully exiting those applications is the reliable migration path. Stable-loader lifecycle tests do not establish real-core cross-version handover in every editor. See [Windows validation](../quality/windows-validation.md) for measured acceptance and remaining gates.

Windows is an early preview. Cloud polishing is a separate, explicit choice ([ADR 022](../decisions/022-cloud-polishing.md)): with a connected provider and selected model it sends final text, scene and recalled terms only. Audio, cursor/window context and dictation history remain on-device. Local remains the default; no cloud request is made for that engine. Real ChatGPT sign-in and provider requests on supported hosts remain unaccepted (CLD-A05); cloud quality and latency have not been benchmarked. Model weights are downloaded separately from the pinned catalog and are not embedded in installers.

The macOS package requires macOS 14 or later. Speech recognition now packages native Qwen3-ASR workers and the Swift/Metal resource closure ([ADR 020](../decisions/020-native-qwen-asr.md)); rewriting stages llama.cpp ([local rewrite](local-rewrite.md)). Package verification rejects any executable whose minimum exceeds the advertised package minimum. Minimum-version host acceptance and minimum RAM measurements remain open.

The latest native arm64 actual-engine/AppKit run passes 36 cases. The translated x86_64 run passes 36 cases under Rosetta 2 on arm64 hardware, not physical Intel. Both use isolated synthetic learning keys; these harnesses do not exercise installed IMK dispatch. A separate user-assisted signed installation subsequently passed physical Shift switching and Right Option dictation in the reported Codex/Cursor editors, including a speech session committed to its target. Duplicate host modifier events and a stale Input Monitoring entry were diagnosed and repaired; see the [incident report](../quality/keyboard-shortcut-incident.md) for evidence and limits. This local acceptance does not establish clean-machine installation or every editor.

Earlier acceptance attempts on macOS 26.6.2 observed an enabled child but disabled parent source, with selection returning `-50`, and restored the original bundle without reaching typing dispatch. Those historical failed attempts remain in the [alignment report](../research/imk-typing-alignment.md); packaging trust alone did not resolve source activation.

## Signing and update trust

The package identifiers are `si.astrafoundry.arion` and `si.astrafoundry.inputmethod.arion`; the IMK input-source ID is `si.astrafoundry.inputmethod.arion.pinyin`. Changing from development identifiers requires a fresh permission grant and removal of the old input source through the older build before testing the new one. Local user data paths have not been migrated.

Ordinary macOS builds use ad hoc signing. Complete Apple environment configuration, including the matching IMK Developer ID provisioning profile, enables Developer ID signing and notarization through the same build entry point; tagged CI builds require it. A retained macOS arm64 canonical build before the strict Keychain policy/profile gate passed Developer ID signatures, hardened runtime, notarization tickets and package Gatekeeper validation. Signed input-method dispatch has since passed on the development Mac; clean-machine acceptance remains open. Automated orchestration tests alone do not establish these host outcomes. Windows publisher signing remains unimplemented. Updating an ad hoc build can require granting privacy permissions again. A valid ad hoc signature does not establish Gatekeeper approval.

Tagged releases require a matching Ed25519 update key and Apple signing/notarization credentials, and verify every platform's archive size and signature before upload. The update key is independent of Apple certificates. Without release credentials, ordinary CI still produces installer artifacts; the release job fails before creating a draft. CI stages verified sets in release storage and a GitHub draft; neither is offered by the updater. Prereleases are offered only on the `beta` channel, which requires the persisted Allow beta updates opt-in. Publishing a channel in the AstraFoundry admin remains a manual decision after acceptance ([ADR 019](../decisions/019-center-update-channels.md)).

## Acceptance before broad distribution

- Run real microphone capture, recognition and insertion on clean macOS and Windows systems, including offline use after provisioning, cancellation, focus changes and device loss.
- Exercise input-source install/select/remove and composition cleanup in owned host applications before claiming IME support.
- Verify install, uninstall and an old-to-new published update on each supported platform; check preferences, vocabulary, credentials and consent, plus a failed replacement.
- Validate minimum OS/RAM and the supported editor compatibility matrix on actual hosts. Native shell E2E and compile checks do not replace these runs.
- Complete held-out meaning/style review before making polishing quality claims. The model catalog preserves conversion and evaluation limitations; shipping its provenance does not resolve them.

Detailed evidence and remaining limits are in [validation](../quality/validation.md) and [desktop release gates](deployment.md#desktop-release-gates).

The [Windows validation](../quality/windows-validation.md) records actual native interaction, microphone lifecycle, installed-app speech delivery, installer/uninstaller and isolated updater checks. These pass on the documented Windows 11 host; production signed-channel upgrade and broad editor/device acceptance remain open.

Windows packaging rejects unbundled runtime DLL imports and checks the CPU worker with a system-only search path. This catches compiler-library dependencies on build machines; it does not replace installation and device acceptance on a clean end-user machine.

The strict keyboard learning provider now requires the Data Protection Keychain and rejects legacy keys without changing their vaults. Native arm64 and Rosetta regressions each pass 23 cases after this change, and real owned legacy-key rejection restores both stores to absence. A Developer ID certificate alone returned missing entitlement (`-34018`); a matching profile is now available and a profile-authorized signed harness passes the real six-phase lifecycle with verified device-only/nonsynchronizing OS attributes and cleanup. The current canonical app/DMG completed signing, Accepted notarization, profile/entitlement validation and Gatekeeper/package checks. Earlier signed artifacts predate this protected-key verification ([deployment](deployment.md), [v5 evidence](../../benchmarks/runs/imk-keyboard-v5/README.md)).

The [100-case development baseline](../../benchmarks/results/imk-quality-v1/README.md) records 71% exact intended commit and 13.43% character error rate. These engine measurements do not establish competitor parity, population accuracy, actual IMK key-to-preedit latency or concurrent speech behavior.

The [v7 evidence archive](../../benchmarks/runs/imk-keyboard-v7/README.md) preserves the literal-umlaut failure and corrected 23-case native/translated runs, rebuilt unsigned bundle checks, the inconclusive inactive-client candidate probe and bounded Doubao UI-refresh follow-up.

The [native ASR migration acceptance](../../benchmarks/results/native-asr-v1/README.md) replaces the old speech stack. Native MLX loading, real-time replay and offline signed-worker inference pass; first-partial and final-latency budgets remain failed. Windows development-host runtime and installed speech checks now pass as documented above; minimum-OS and clean-editor acceptance remain open. Intel macOS is retired from future product support; its existing evidence remains historical.
