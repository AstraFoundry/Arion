# Desktop release status

This file accompanies each release as `RELEASE-STATUS.md`. The `0.1.0-alpha.5` release distributes only macOS Apple silicon packages. The other columns below describe implementation status, not assets included in this alpha. Build success verifies packaging; the evidence below limits the supported product claims.

| Capability | macOS Apple silicon | macOS Intel | Windows x64 |
| --- | --- | --- | --- |
| Native package | App and DMG | App and DMG | Per-user NSIS installer |
| Local speech recognition | Native MLX 1.7B 8-bit; 118 development fixtures measured | Native speech.cpp Q8_0 CPU executable cross-built; actual Intel inference pending | Native CPU/Vulkan staging implemented; actual Windows package/inference/signing acceptance pending |
| Local polishing | Worker and pinned runtime bundled | Unsupported; no approved runtime/model pair | Unsupported; no approved runtime/model pair |
| System input method | Offline Pinyin/voice IMK host bundled with its own horizontal candidate bar and context ranking; native 28/28 cases pass; signed source selection still fails | x86_64 23/23 cases pass under Rosetta on arm64; physical Intel/system dispatch untested | TSF not implemented |
| Encrypted sync | Desktop enrollment and outbox pending | Desktop enrollment and outbox pending | Desktop enrollment and outbox pending |

Windows is an early preview. Polishing on unsupported platforms reports its unavailable state; no content is routed to cloud inference. Model weights are downloaded separately from the pinned catalog and are not embedded in installers.

The macOS package requires macOS 14 or later. Speech recognition now packages native Qwen3-ASR workers and the Swift/Metal resource closure ([ADR 020](../decisions/020-native-qwen-asr.md)); rewriting stages llama.cpp ([local rewrite](local-rewrite.md)). Package verification rejects any executable whose minimum exceeds the advertised package minimum. Minimum-version host acceptance and minimum RAM measurements remain open.

The latest native arm64 actual-engine/AppKit run passes 28 cases. The translated x86_64 run passes 23 cases under Rosetta 2 on arm64 hardware, not physical Intel. Both use isolated synthetic learning keys. These isolated runs do not exercise installed IMK event dispatch, an external editor or the user's Keychain. A completed macOS arm64 Developer ID signed/notarized desktop build passed package Gatekeeper checks, and the exact copied nested host was accepted as Notarized Developer ID after installation. Nevertheless, on macOS 26.6.2 its parent source remained disabled while its child mode was enabled: native enable returned `0`, but select returned `-50`. No actual typing dispatch was reached. Run 05 restored the original bundle after that failed probe. A later final-app UI attempt observed onboarding/menus but never reached installation; it ended with ABC, no own-source entries, original bundle bytes and absent keyboard key restored without errors. Source activation is still unaccepted, and signing alone does not explain or resolve the failure ([alignment report](../research/imk-typing-alignment.md)).

## Signing and update trust

The package identifiers are `si.astrafoundry.arion` and `si.astrafoundry.inputmethod.arion`; the IMK input-source ID is `si.astrafoundry.inputmethod.arion.pinyin`. Changing from development identifiers requires a fresh permission grant and removal of the old input source through the older build before testing the new one. Local user data paths have not been migrated.

Ordinary macOS builds use ad hoc signing. Complete Apple environment configuration, including the matching IMK Developer ID provisioning profile, enables Developer ID signing and notarization through the same build entry point; tagged CI builds require it. A retained macOS arm64 canonical build before the strict Keychain policy/profile gate passed Developer ID signatures, hardened runtime, notarization tickets and package Gatekeeper validation; clean-machine acceptance and signed input-source activation remain open. Automated orchestration tests alone do not establish these host outcomes. Windows publisher signing remains unimplemented. Updating an ad hoc build can require granting privacy permissions again. A valid ad hoc signature does not establish Gatekeeper approval.

Tagged releases require a matching Ed25519 update key and Apple signing/notarization credentials, and verify every platform's archive size and signature before upload. The update key is independent of Apple certificates. Without release credentials, ordinary CI still produces installer artifacts; the release job fails before creating a draft. CI stages verified sets in release storage and a GitHub draft; neither is offered by the updater. Prereleases are offered only on the `beta` channel, which requires the persisted Allow beta updates opt-in. Publishing a channel in the AstraFoundry admin remains a manual decision after acceptance ([ADR 019](../decisions/019-center-update-channels.md)).

## Acceptance before broad distribution

- Run real microphone capture, recognition and insertion on clean macOS and Windows systems, including offline use after provisioning, cancellation, focus changes and device loss.
- Exercise input-source install/select/remove and composition cleanup in owned host applications before claiming IME support.
- Verify install, uninstall and an old-to-new published update on each supported platform; check preferences, vocabulary, credentials and consent, plus a failed replacement.
- Validate minimum OS/RAM and the supported editor compatibility matrix on actual hosts. Native shell E2E and compile checks do not replace these runs.
- Complete held-out meaning/style review before making polishing quality claims. The model catalog preserves conversion and evaluation limitations; shipping its provenance does not resolve them.

Detailed evidence and remaining limits are in [validation](../quality/validation.md) and [desktop release gates](deployment.md#desktop-release-gates).

The strict keyboard learning provider now requires the Data Protection Keychain and rejects legacy keys without changing their vaults. Native arm64 and Rosetta regressions each pass 23 cases after this change, and real owned legacy-key rejection restores both stores to absence. A Developer ID certificate alone returned missing entitlement (`-34018`); a matching profile is now available and a profile-authorized signed harness passes the real six-phase lifecycle with verified device-only/nonsynchronizing OS attributes and cleanup. The current canonical app/DMG completed signing, Accepted notarization, profile/entitlement validation and Gatekeeper/package checks. Earlier signed artifacts predate this protected-key verification ([deployment](deployment.md), [v5 evidence](../../benchmarks/runs/imk-keyboard-v5/README.md)).

The [100-case development baseline](../../benchmarks/results/imk-quality-v1/README.md) records 71% exact intended commit and 13.43% character error rate. These engine measurements do not establish competitor parity, population accuracy, actual IMK key-to-preedit latency or concurrent speech behavior.

The [v7 evidence archive](../../benchmarks/runs/imk-keyboard-v7/README.md) preserves the literal-umlaut failure and corrected 23-case native/translated runs, rebuilt unsigned bundle checks, the inconclusive inactive-client candidate probe and bounded Doubao UI-refresh follow-up.

The [native ASR migration acceptance](../../benchmarks/results/native-asr-v1/README.md) replaces the old speech stack. Native MLX loading, real-time replay and offline signed-worker inference pass; first-partial and final-latency budgets remain failed. Windows/Intel/minimum-OS and clean-editor acceptance are still open.
