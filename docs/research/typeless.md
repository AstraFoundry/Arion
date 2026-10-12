# Typeless installed-artifact investigation

## Target and scope

Installed `/Applications/Typeless.app`: version **2.8.1**, build **2.8.1.130**, bundle ID `now.typeless.desktop`. The app contains Electron, `Resources/app.asar`, native FFI helpers and packaged SQL migrations. Artifact digests and bounded offsets are in the [ledger](evidence/typeless-static.json).

REA CLI 6.4.0 actually produced application Evidence `ev_17708dd83c71fab9ced0d18a4df9036551e5e4aca46326f121f929f1cbaa2ccb`. Its summary-view ID is recorded in the ledger. No REA native session was opened, so there is no session to close. The active chat lacked connected REA tools; a read-only doctor found aligned Codex registration but no configured native decompiler. No client configuration or engine was installed.

The first ASAR analysis rejected a native companion digest mismatch. The next call explicitly used `record-and-continue`, preserving both contradictions as **observed-untrusted**. This does not establish malicious tampering or byte equivalence: signing/stripping is one possible explanation, not verified here. Full immutable analysis and view remain in ignored `.workspace/research/`; the committed ledger is a compact derivative, not replayable full Evidence.

## Coverage and limits

REA inventoried 10,499 relevant files, parsed 6,007 JavaScript files and recorded three parse failures. Application graph coverage is **partial**, with budget-constrained expansion and unresolved dynamic/obfuscated relationships. Complete file inventory does not imply complete behavior recovery. Native helper exports, cloud model choices, actual requests, server retention and runtime reachability are not established by this analysis.

The six module identities in the ledger come from REA; TL-* literal observations are separately obtained from inert packaged bytes. No bundled bootstrap code was executed. User recordings, private history databases, credentials and live traffic were not examined.

## Pipeline evidence

| Stage | Direct static evidence | Interpretation and boundary |
| --- | --- | --- |
| Trigger and OS bridge | Packaged keyboard helper, shortcut names and native callback surfaces | Global shortcut/native integration exists in the package; physical key semantics and callback ordering unverified |
| Target context | `getFocusedAppInfo`, `text_insertion_point`, `cursor_state`, selected-text fields; context-helper library | Structured application/cursor context feeds the client data model; exact AX traversal and sent fields require native/runtime evidence |
| Microphone | Renderer uses `navigator.mediaDevices.getUserMedia` with audio device options | Browser audio capture path is present; runtime chosen device/rate not measured |
| PCM capture | Custom AudioWorklet processor; Float32 buffers; 1,024-sample message batches; explicit stop flush | Capture batches PCM before processing; 1,024 is a code-level batch size, not proof of end-to-end latency |
| Compression/session | Chunk-session worker with native session functions; Opus chunk and Opus encoding libraries/workers | Native audio chunk/compression machinery is present; compression mode per session remains unverified |
| Remote stream | Renderer constructs WebSocket, installs open/close handlers and carries audio-session/chunk control names | Supports an inferred streaming service boundary; static presence is not observed traffic or proof of a particular server protocol |
| Refinement | Raw/refined fields, audio-processing lifecycle events and refine-completion handlers | Transcript and polished text are separate client states; exact backend ASR/LLM ordering/model is unknown |
| Delivery | Insert-text IPC name, input-helper library, target/cursor data and insertion symbols | Native delivery bridge is present; clipboard vs AX path selection and insertion success unverified |
| Persistence | Packaged SQL schema includes refined text, audio/context metadata and local audio path | A local history storage design exists; no user's database was read and schema presence does not prove every field is populated |
| Memory/history sync | Dictionary settings, user-edit surfaces, sync-state store and history push/pull/status routes | Separate personalization/history synchronization surfaces exist; their runtime defaults, conflict logic and encryption are unknown |

## Reconstructed flow

```mermaid
flowchart LR
  K[Shortcut / native keyboard bridge] --> C[Focused target and context snapshot]
  K --> M[Renderer microphone capture]
  M --> W[AudioWorklet PCM batches]
  W --> O[Chunk session / Opus worker surfaces]
  O --> N[WebSocket / remote processing boundary]
  C --> N
  N --> R[Transcription and refined result state]
  R --> I[IPC / native input helper]
  I --> T[Target editor]
  R --> H[Local history schema]
  H --> S[Optional history sync surfaces]
```

This is a static reconstruction with inferred edges, not a traced call graph. It does not imply every session takes every branch or that ASR/refinement both run on the same server/model.

## Public documentation corroboration

Typeless's current policy describes cloud processing of voice and limited app/text context, with optional history synchronization. That is a **vendor statement**, not proof of installed-client enforcement or server behavior. [Official privacy policy](https://www.typeless.com/privacy). Its personalization guide describes abstract style learning rather than retaining specific writing content; the actual algorithm is not established. [Official personalization guide](https://www.typeless.com/help/quickstart/personalization).

Do not equate local history or a zero-retention statement with local speech inference. Arion's offline inference boundary is a deliberate different design choice.

## Checklist for this investigation

- **Answered:** installed version/package/entrypoint identity; microphone/worklet source; native context/input/compression surfaces; history schema and sync route presence.
- **Partially answered:** trigger-to-capture-to-result-to-delivery flow; refinement/personalization stage division. Static source provides boundaries but obfuscation/native/backend edges remain unknown.
- **Unresolved:** exact ASR/correction server models, complete protocol/call ordering, actual latency/traffic/retention, native helper traversal/insertion semantics and runtime sync policy.

Arion should reuse the separation of capture, draft/refinement and target delivery as an architectural lesson. It should implement its own ownership/revision guards, local models and approved-memory protocol rather than copying commercial bundled code or treating unresolved server behavior as a requirement.
