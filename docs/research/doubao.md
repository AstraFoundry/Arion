# Doubao IME: local artifact investigation

## Scope and evidence status

Target: `/Library/Input Methods/DoubaoIme.app`, installed bundle version **1.0.2**, build **1**. This report describes the installed macOS artifact, not the latest release or Windows/mobile implementation.

Two passes exist. The first used read-only Python plist/byte parsing and the system `file`, `otool` and `nm` tools; `DS-*` identifiers are locally assigned observations in [doubao-static.json](evidence/doubao-static.json) (SHA-256 identities, byte offsets, imports, resource headers, reproduction commands). The second used the `reverse-engineer-anything` skill's REA 6.4.0 CLI with the Ghidra 12.1.4 provider on the arm64 slice of the main executable and produced REA Evidence IDs; `DD-*` identifiers in [doubao-decompiled.json](evidence/doubao-decompiled.json) map to them. See [Decompiled insertion and shortcut paths](#decompiled-insertion-and-shortcut-paths).

Only application-package files were read. The app was not launched or attached; no microphone recordings, user databases, runtime logs, account material, or network interception were accessed. Consequently, **execution order, active settings, latency, transmitted payloads, model weights, and server implementation are unresolved**.

## Finding ledger

| Question | Status | Observation and interpretation |
| --- | --- | --- |
| How does it integrate with macOS text input? | Answered for package integration; runtime partially answered | `DS-identity` registers `DoubaoImeInputController`, `DoubaoIme_Connection`, and a simplified-Chinese pinyin input source. `DS-links` imports `IMKInputController` and loads InputMethodKit. This establishes an IMK input-method package. |
| How does audio reach recognition? | Partially answered | `DS-capture` contains `ASRAudioEngine`, `ASRManager`, PCM conversion and framed-Opus names. `DS-links` imports `AVAudioEngine` and loads AVFoundation/CoreAudio. Capture and audio conversion components exist; their sample rate and actual frame/codec selection were not recovered. |
| Is recognition cloud-only? | Partially answered | `DS-online_offline` contains SAMICore WebSocket pool/TTNet names, `ExecuteOnlyOfflineTask`, `ExecuteSwitchTask`, and a diagnostic for an empty model path choosing online ASR. Both online and offline implementation surfaces are present. Default policy and runtime selection remain unknown. |
| What runs locally? | Partially answered | `DS-local_kernel` contains Panther/Qwen3 decoder names, W4A8 transformer kernels, and CPU/Metal provider diagnostics. `DS-links` also loads ONNX Runtime 1.16.2. This supports local neural inference capability, but does not identify a deployed model, parameter count, architecture topology, or active runtime. |
| How are revisions displayed? | Partially answered | `DS-results` names complete and recent VAD results, full-VAD and offline flags, and VAD modification pairs. `DS-links` imports a text-display `applyDiff(deleteCount:insertText:)` method. `DS-commit` retains marked-text selectors. This is compatible with revisable partial display plus final commit, without establishing the exact revision algorithm. |
| How are context and corrections handled? | Partially answered | `DS-context` contains AX selection/text markers, hotword context, a global ASR context manager, NER/request types and upload-request names. `DS-correction` contains ASR user-correction, NER, n-gram and character-error dictionary names. Local context/dictionary machinery exists; actual collection, upload, account association and sync semantics are unknown. |
| Is final delivery a single text insertion? | Partially answered | `DS-commit` contains final-commit transaction/ownership/receipt types, IMK insert selectors, synthetic Command-V diagnostics and pasteboard restoration guards. Several host-delivery routes and lifecycle guards exist; their routing conditions are unresolved. |
| Does it support semantic rewriting? | Partially answered | `DS-organize` names a smart-organize client, cache and undo snapshots. Public product descriptions advertise text organization. Endpoint, prompt, model location and availability in offline mode were not established. |

## Candidate pipeline

The following ordering is an **architectural inference from component names and interfaces**, not a recovered callgraph or observed recording session:

```text
Shortcut / input-controller entry
  -> capture host/context and establish recording session
  -> audio capture / PCM conversion / optional encoded transport
  -> selected recognition mode
       online: SAMICore WebSocket / TTNet connection pool
       offline: installed model files / local inference providers
       switching: ExecuteSwitchTask surface; policy unknown
  -> VAD-oriented result updates and modification pairs
  -> display diffs / IMK marked-text composition
  -> punctuation policy and optional smart organization
  -> target/ownership validation and final commit transaction
       IMK insertText route or synthetic paste route
  -> completion receipt, cleanup, guarded clipboard restoration
```

### Native text input and host compatibility

The installed input method and its nested `DoubaoImeSettings.app` are distinct bundles. The main bundle is background-only; settings is a UI-element app. The main executable is universal x86_64/arm64. Shared framework linkage and IPC/offline-model notification names support separation of settings/model management from input-controller responsibilities (`DS-identity`, `DS-online_offline`). IPC transport was not decoded.

`insertText:replacementRange:` and `setMarkedText:selectedRange:replacementRange:` coexist with accessibility text-marker and selection names. Diagnostics name target ownership rejection, delayed paste cancellation, a latest result pending while paste is in flight, and avoiding clipboard restoration after the clipboard changes (`DS-context`, `DS-commit`). These are shipped **diagnostic templates**, not events observed in logs. They make stale results, focus changes and asynchronous insertion useful design concerns for Arion; they do not prove Doubao's guards work in any particular host application.

### Recognition and local models

The package includes downloadable-model management surfaces: manifest, active-layout store, path resolver, storage-health checks and an unavailable-model prompt. `asr_model.model`, `offline_model_manifest.json`, `context_aux_model`, and version-directory names are static literals (`DS-online_offline`, `DS-results`). This investigation did not read downloaded resources outside the shipped bundle and cannot say whether a local ASR model is installed on this device.

The main binary's `W4A8ImeAsrQwen3Transformer` and CPU/Metal diagnostics establish compiled support for those kernel paths. They **do not prove that a public Qwen3-ASR checkpoint is used**, or that all transcription/correction runs on that path. ONNX linkage likewise does not establish which features use ONNX rather than the embedded Panther stack (`DS-local_kernel`, `DS-links`).

Three shipped files under `OimeEngineResources.bundle/asr/` have SHA-256 identities and 24-byte prefixes retained in `DS-resource-headers`. The `file` tool identifies them only as `data`; their format remains unknown. No verified ONNX/model parser was applied. The `asr` directory name alone cannot establish whether these opaque resources are acoustic models, dictionaries, metadata, or another payload.

### Context, correction and organization

The result surface distinguishes a complete result, recent VAD result, a full-VAD flag, offline-result flag and `VADModifyPair` collections (`DS-results`). These names are compatible with segment corrections and finalization boundaries. Whether modification pairs come from server postprocessing, local recognition, dictionary matching or another source remains unknown.

OimeEngine names user-correction, user-NER, n-gram, exempt-word and character-error dictionaries (`DS-correction`). The shipped ASR context manager also includes `TextNERRequest`, `UserWordsUploadRequest`, `ModifyPairUploadRequest`, and `AccountAwareContextRequest` type names (`DS-context`). Those names establish implementation surfaces only; they do not demonstrate current uploads, cross-device memory sync, retention policy, or payload contents. In particular, clipboard transfer advertised publicly is a different feature from synchronized contextual memory.

Smart organization has its own client/cache/undo types (`DS-organize`). Treat lightweight recognition correction, punctuation formatting and semantic organization as separate capabilities when designing Arion. An offline ASR engine does not by itself establish an offline organization pipeline.

## Decompiled insertion and shortcut paths

The executable is stripped, but Ghidra recovered Swift stored-property offset symbols of `DoubaoImeInputController` and Objective-C selectors, which make decompiled control flow legible. REA could not link Swift string literals to code (trace and xrefs returned no references, `DD-negative-references`), so code sites were located by scanning for the instructions that compute the encoded literal addresses; a decompiled body was used only when it contains the literal. Five functions were decompiled; four are interpreted below.

| Question | Status | Observation (REA Evidence) |
| --- | --- | --- |
| What gates a delayed paste? | Answered for this function | `DD-paste-gate` (`ev_7813c9cf…`): Command-V is posted only if the session, a final-result generation counter and a paste commit sequence still match the captured values, a paste is in flight, Command-V has not been posted and the final-text write count is zero; otherwise `staleBeforeCommandV`. Further named failures: target changed before post, pasteboard changed during write, pasteboard write or marker failure, preferred target changed (before or after a hook). A posted paste is logged with `hostAck=unknown`. |
| Host-specific handling? | Observed for one host | `DD-paste-gate`: when the host bundle identifier is `com.microsoft.Excel` and a mouse button is held (`NSEvent.pressedMouseButtons`), the commit waits for mouse release before pasting. |
| How is the clipboard restored? | Answered for this function | `DD-paste-handoff` (`ev_f83fcb7d…`): a restore token is incremented and the scheduled restore work item cancelled; the current `changeCount` is compared with the count recorded at write. Equal hands the pasteboard off; different skips with `changedAfterPaste`. |
| Can the host commit on its own? | Partially answered | `DD-commit-strategy` (`ev_3af76376…`): a host commit resolution (`hostCommitted`, `imeCommitRequired`, `ambiguous`, `contextInvalid`) is evaluated first; when the host already committed the original text the strategy is `skipSystemCommitted` with zero writes. A host text-implementation classification (`web`, `qt`, …) is read in the same step. |
| Hold-to-talk? | Partially answered | `DD-shortcut-fn` (`ev_2ad00740…`): a long-press shortcut profile exists beside the single-click one; `longPressUsesFn` is true for key code 0x3F or a modifier-only shortcut whose flags equal `NSEventModifierFlagFunction`. Shipped messages promote a pending long press to a single click when the panel is opening. The capture mechanism and threshold were not recovered. |

Unresolved: the routing condition between direct `insertText` and paste (its function entry was not isolated), the stale offline-model callback path, the `OimeCommon`/`OimeEngine` frameworks (including the text-display `applyDiff`, `DD-text-display`), and any runtime ordering or host behavior.

## Public-source corroboration

The [official desktop product page](https://ime.doubao.com/pc) describes dialect and mixed Chinese/English voice input, keyboard prediction/correction, optional text organization, and cross-device clipboard transfer. These are vendor descriptions, not measurements of the installed package. The page was retrieved successfully; its claims do not identify model internals or memory-sync behavior.

The [IME-specific privacy policy](https://lf3-cdn-tos.draftstatic.com/obj/ies-hotsoon-draft/wave_ime/ime_privacy_policy.html) indexed excerpt states that downloaded, enabled offline voice resources process speech locally without uploading that input. Direct page retrieval timed out during this investigation; this corroboration is limited to the indexed vendor-policy excerpt and was not independently verified by a recording or traffic capture. It does not justify a claim that every feature or default mode is offline. The general Doubao chat-app policy was excluded because it concerns a different product.

## Implications for Arion

These are proposed product/architecture decisions, not copied competitor implementations:

- Keep native IME integration distinct from the settings shell. MyGo hosting alone does not establish IMK or Windows TSF support.
- Model recognition updates as revisable hypotheses with segment/session identities and explicit finalization. Keep committed output separate from mutable composition.
- Capture only authorized bounded context; hotword biasing and persistent user memory need separate contracts and privacy rules.
- Own final insertion as a transaction tied to host, selection and session. Reject stale results and focus changes; return an explicit delivery outcome. Doubao's decompiled paste gate checks session, generation, commit sequence and a zero write count immediately before Command-V and reports an unknown host acknowledgement; Arion's coordinator generation check, `ClaimCommit` and `unknown` outcome cover the same failure modes.
- Restore the clipboard only if nothing else changed it (Doubao compares `changeCount`; Arion's clipboard route does the same).
- Offer hold-to-talk as a separate shortcut profile, including the fn key, distinct from toggle; Doubao ships both.
- Expose offline recognition readiness and failure explicitly. Do not silently switch a privacy-sensitive local session to cloud recognition.
- Evaluate lexical correction, punctuation and semantic rewriting separately, including the privacy and latency of each stage.

## Remaining investigation

Decompilation answered the paste gate, clipboard handoff, host-commit resolution and fn long-press profile for the functions listed above. A bounded runtime study or further decompilation is still needed for stage ordering, sample rate/codec, default mode, online/offline switching, retry policy, partial cadence and the direct-versus-paste routing condition. A verified model manifest/parser is needed for downloaded model identity and licensing. Windows, iOS and Android binaries were outside this local target. No conclusion here claims those questions are resolved.
