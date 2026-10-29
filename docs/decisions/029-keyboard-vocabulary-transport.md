# 029: Shared vocabulary delivery over verified input-method transport

## Status

Accepted. Application protocol, macOS mutual publisher authentication, native dispatch, snapshot refresh/retry and reading UI are implemented. The PR 45 execution records linked below describe that revision. The integrated PR 40 implementation has separate validation; historical signed or installed results do not establish current integrated-binary acceptance. Full app workflow, installed restart and Windows delivery remain separate gates.

## Context

MEM-08 and KBD-11 reuse approved encrypted vocabulary as a keyboard candidate source. The native encrypted projection is already separate from ordinary keyboard learning. Delivering private terms needs a verified peer, complete-snapshot deletion semantics and restart-safe revision negotiation. Existing macOS protocol-0 admission does not provide the process proof required by the new operation.

## Decision

Extend the existing local IME connection with an optional `keyboard-vocabulary-v1` hello capability. A capability claim is not authentication: publication requires the existing verified peer identity and a live process lease. Older hosts retain voice behavior and receive no vocabulary requests. macOS uses actual OS-backed peer verification before enabling delivery; Windows retains ADR 028 admission rules.

Only one vocabulary transaction may be pending per bridge. It is independent of voice revision/session identifiers, and is bound to the accepted connection, active editor and monotonically increasing request number. Connection replacement, deactivation, update installation and voice capture invalidate it. A caller timeout or rejection leaves desired application state pending for an explicit retry.

The wire exchange uses the existing newline-delimited JSON framing:

1. App sends `vocabulary_query` with `request` and `activation`.
2. Host returns `vocabulary_state`, echoing both, with `status` and `vocabularyRevision` from validated persisted state.
3. App sends `vocabulary_publish` with the same request/activation, revision plus one, and explicit `terms` array of `{id,text,reading}`. An empty array removes the complete shared projection.
4. Host returns `vocabulary_published`, echoing the request/activation and newly persisted revision, only after successful persistence and idle-session application.

`status: "ok"` is success, `"busy"` is temporarily unavailable, and any other status is rejection. The app rejects mismatched acknowledgement revisions and ignores responses for other requests, activations or phases. Revisions must not wrap. Retries query persisted state again: a missing acknowledgement does not prove that publication failed to persist.

Validate and order the complete snapshot through the existing domain projection. Bound the JSON frame below one MiB and the transaction to three seconds. Do not export correction aliases or speech counts, copy terms into plaintext settings, open a second listener or modify ordinary learned words. Native handling must independently validate terms against its dictionary and refuse busy or stale publication without partial application.

## Consequences and validation

The application adapter is testable without changing installed input sources. Pipe-fixture tests cover snapshot/deletion encoding, peer/capability admission, stale responses, overlapping requests, focus loss, disconnect, update, cancellation, busy rejection, incorrect acknowledgement and lost-acknowledgement retry. These fixtures do not prove OS peer authentication or native persistence.

The macOS typing adapter also validates message identities, typed bounded term arrays and dictionary-resolvable readings, then calls the existing encrypted publish/restore boundary. Actual Rime/AppKit tests cover successful candidate production, busy and malformed rejection, deletion and restored revision. The host now dispatches it only after mutual publisher authentication; an unverified socket cannot reach this method.

The macOS bridge can now obtain the kernel `LOCAL_PEERTOKEN` from an actual Unix socket and locate the corresponding live `SecCode` through `kSecGuestAttributeAudit`. It enforces equal effective user IDs and checks signing requirements supplied by trusted application code, never by hello messages. A serialized reference lease rejects checks after release; dynamic validity rejects an exited process. Real child-process socket tests verify identity, signing-requirement mismatch, process exit and integration through the bridge listener. The test requirement `always` proves the mechanism only and is not a production publisher policy. Non-cgo builds reject native peer-proof requests.

The production policy derives its publisher from the running binary after validating its Developer ID Application chain. It requires the peer to carry the expected code identifier and the same ten-character team identifier; peer hello fields and environment overrides cannot choose either publisher. The application expects the IME bundle ID. The native host expects the application ID already embedded in its signed Info.plist by the build. Both consumers compile the same C header. Accepted requirements are retained and checked during the connection lifetime. Ad-hoc development binaries cannot acquire publisher authority; independent offline typing is unchanged. The host advertises vocabulary capability and dispatches messages only on an authenticated connection. Older correctly signed hosts without that capability retain voice admission.

Actual [Developer ID process fixtures](https://github.com/AstraFoundry/Arion/blob/264b882fc1f83adfc29285d3e80324f308734671/benchmarks/runs/ime-publisher-v1/README.md) now verify positive mutual publisher binding, wrong-identifier/ad-hoc rejection, process-exit/release invalidation and admission through the production Go bridge. The runner signs owned executables using an available local certificate; production publisher selection remains derived from the running signed process. These fixtures exercise the shared native proof but do not establish installed application/IMK dispatch or cross-process native vocabulary persistence.

The application queues its approved projection after each successful vocabulary load. One bridge worker retains only the latest validated snapshot, deduplicates unchanged terms, cancels superseded attempts and re-delivers on connection/activation changes. Busy or failed attempts retry at one-second intervals; connection absence, voice ownership and updates wait for lifecycle notification. Delivery outcomes are reported for each new snapshot even when the previous snapshot had the same outcome; repeated identical outcomes for one snapshot remain deduplicated. Vocabulary settings show pending application for a saved keyboard edit or deletion, then clear it only on acknowledgement of that snapshot. Native rejection or transport failure shows a localized failure while retaining the saved entries. Main-thread projection identity rejects queued outcomes for older edits; unchanged loads do not clear a failure or restart a pending notice. An initially empty projection has no unrelated notice. Format validation does not claim that the native dictionary accepts a reading. Missing or unreadable application storage is not an empty deletion snapshot. Closing the bridge cancels the worker. Tests cover supersession, busy recovery, same-state deduplication, reconnect, voice/update release and shutdown, plus encrypted-store load/deletion/failure integration.

The opt-in `keyboard:vocabulary:check` extends the signed process runner with the production Go publication API and native `ArionTyping` handler over a real Unix socket. Three separate signed native processes share one synthetic encrypted profile: publish an explicitly bound term and commit it through actual Rime/AppKit, restore and delete it, then restore the empty revision. Both sides enforce the production publisher policy. The fixture reuses the existing native typing build and editor helpers; it does not register an IMK server or use a personal Keychain key. [Archived evidence](https://github.com/AstraFoundry/Arion/blob/264b882fc1f83adfc29285d3e80324f308734671/benchmarks/runs/ime-native-vocabulary-v1/README.md) includes the initial C++ compilation failure and successful runs. This proves isolated signed delivery and persistence, not installed editor dispatch.

The [current full-host package preflight](https://github.com/AstraFoundry/Arion/blob/264b882fc1f83adfc29285d3e80324f308734671/benchmarks/runs/keyboard-p0-package-v1/README.md) also passes the real dual-architecture host build, isolated Developer ID signing, profile authorization, runtime/data closure and production installer preflight. That artifact is neither launched nor installed and has no release timestamp/notarization.

Remaining gates are signed installed application/IMK vocabulary delivery, Windows native execution and installed-host offline/restart acceptance. Isolated positive signed-process mutual authentication is verified separately. Elevated Windows typing continues to operate from its local cache without expanding voice IPC trust.

A newly registered observer for an already acknowledged snapshot receives that acknowledgement without another native publication, provided the admitted connection and active editor still match. This lets a rejected over-budget edit recover to the previous saved snapshot without leaving its new pending notice unresolved. Tests cover acknowledged A, rejected B, restored A and absence of another wire write.

## Alternatives considered

- A second plaintext dictionary file: duplicates ownership and exposes private terms.
- Delete words from ordinary Rime learning: can erase independent keyboard learning.
- Publish to any hello-capable peer: self-reported capability is not process identity.
- Blindly retry the same revision: cannot distinguish a lost acknowledgement from a failed write.

## Windows module boundary

The optional keyboard module exports revision-returning restoration and full-snapshot publication through its existing serialized engine boundary. The host requires both symbols before accepting the module and uses the same shared personal-vocabulary validation and encrypted persistence as macOS. The module fixture covers fresh revision, stale publication, composition-busy rejection, restoration into a new session and explicit empty-snapshot deletion. MinGW cross-target syntax checks pass for the module, fixture and service; native Windows execution remains required. The wire parser now reuses the strict bounded settings/manifest JSON reader, with separate shared Unicode/quoting helpers rather than a second parser. Vocabulary messages require positive lossless integer request/revision fields, an explicit array (including empty deletion), bounded UTF-8 fields without embedded NUL, and at most 2,000 terms. The existing transport generation and activation gates precede COM-apartment dispatch. Voice ownership and keyboard composition return busy; successful publication returns the persisted revision. The service advertises the capability only with a loaded keyboard module. These source and compilation checks do not establish actual Windows delivery or installed-host acceptance.

The [installed vocabulary run](https://github.com/AstraFoundry/Arion/blob/264b882fc1f83adfc29285d3e80324f308734671/benchmarks/runs/keyboard-installed-vocabulary-v1/README.md) uses the actual signed IMK main dispatcher and production Go vocabulary publisher. Synthetic addition, deletion and re-addition receive successful acknowledgements; real abbreviated keys select and commit the term once. Deletion removes it from the observed first page, and Escape adds no insertion. This verifies neither all-page absence nor installed restart persistence. Original installed bytes and selected source were restored.
