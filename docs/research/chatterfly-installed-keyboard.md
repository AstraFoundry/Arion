# Installed Chatterfly keyboard follow-up

## Result and scope

The installed macOS **Chatterfly 1.0.6.13398 (13398)** has the same executable and common packaged resources as the [earlier installer analysis](keyboard-ime.md#chatterfly). This follow-up recovers concrete model loaders and enablement rules, checks model provisioning and a stored cloud preference, and observes synthetic keyboard composition/edit/cancel/commit in TextEdit.

The installed keyboard converted `nihao` plus Space into **你好**. Its stored cloud-candidate preference is enabled. No known neural keyboard-model file was found in the bounded package/support/cache inventory. Those observations establish a working keyboard and an enabled preference; they do not identify which model or service produced a particular candidate.

The [evidence ledger](evidence/chatterfly-installed-keyboard.json) records artifact hashes, 44 gate functions, nine model/path functions, configuration-key bytes, the bounded asset search and ordinary UI observations. New static functions use byte-verified Apple LLVM instructions, not REA pseudocode. The REA signature inspection has its own Evidence ID; the unsuccessful native UI request has none. Full records/scripts remain in ignored `.workspace/research/chatterfly-installed/`.

## Installed identity

The main executable is 112,951,952 bytes, SHA-256 `1063fbb5b0e18dc323ec4a14487456795b359b9330cfeee14b78454a00791e50`. All **427 common bundle entries** match the retained extracted package. The installed bundle additionally has four framework executable symlinks omitted by extraction. Prior matching arm64 functions can therefore be reused for this installed artifact.

REA `inspect_signature`, Evidence `ev_c16ca1847e7ddc1ad168e45d2a39eb83358507515eaea67a609091e01a02596b`, identifies the signed, hardened input-method executable. This is signature/artifact evidence, not execution evidence for its keyboard algorithms.

## Concrete model roles

| Component | Recovered implementation | Deployment evidence |
| --- | --- | --- |
| Local language-model scoring | `LoadLocalLanguageModel` builds `lanmodel.xnet`, checks existence, calls `LocalLanguageModel::AttachModel`, then `AttachModelXNet`, which constructs `InferenceEngineXNetLMIncremental` | Loader and inference-family capability established; matching model file and successful attachment not observed |
| Feature sorting model | `LoadSortFeatureModelLm` separately loads `model.lm.xnet` through `t_sortFeatureModel::AttachModel` | Separate loader established; matching file not found |
| Generative candidates | Encoder/decoder XNet and SentencePiece filenames from the earlier generation-loader path, plus conditional asynchronous generation | Capability and gate rules established; assets and active execution unresolved |
| Packaged opaque dictionary/LM data | `ime_lm.bin`, `ime_em.bin`, `ime_pyslm.bin`, `ime_gd_bigram.bin` and related resources | Installed bytes match the earlier package; complete proprietary formats remain undecoded |

`CF-installed-models-lm-loader`, `CF-installed-models-lm-attach` and `CF-installed-models-lm-xnet` contain the concrete XNet constructor chain. This narrows the runtime family to an incremental XNet LM implementation. It does not establish an RNN/Transformer architecture, parameter count, public checkpoint or trained weights. The TNN attachment function in this inspected build returns false.

Model-directory helpers select either the engine user directory or a versioned directory beneath `models/`, using model version and, for generation, device grade. `lanmodel.xnet` and `model.lm.xnet` are distinct from the opaque bundled `ime_lm.bin`; a shared filename fragment is insufficient to identify them as the same model.

## Enablement is configuration and readiness dependent

The generation enable field starts **false** in the `t_parameters` constructor. The named configuration lookup `kDeviceModelEnableState` has default **0**. Updates copy requested mode into applied mode and enable the model families as follows:

| Applied numeric mode | XNet local LM predicate | Generation enable predicate |
| --- | --- | --- |
| 4 | True | False |
| 5 | True | True |
| 6 | False | True |
| Other values | False | False |

These are recovered predicates, not recovered enum names or this Mac's observed active mode. A true predicate permits further checks; it does not prove successful model attachment or inference.

Generation additionally checks an unnamed suppression field, readiness, attached dictionaries, device/model state and trigger conditions. `kGenerateModelLoadStatus` and `kGenerateModelUseStatus` separately default to 0, respectively allowing loading and reporting a healthy status. Those defaults do not override the disabled generation-enable field. A loader, an allowed-load status and a ready, executing model are separate states.

The ledger's `staticEvidence.gateFunctions` and `selectedCoreConfigKeys` retain the getter, constructor, update functions and actual pointer/cstring mapping for those keys. No active remote assignment, suppression-field meaning or complete device-threshold semantics were recovered.

## Model provisioning search

The scan recursively inventoried the installed bundle and **ten Chatterfly-specific Application Support/Caches roots**, including `~/Library/Application Support/Chatterfly/InputMethod/ChatterflyPY`. It searched 20 exact filenames covering XNet, SentencePiece, feature-sort, XGBoost and TNN resources, plus common model extensions. There were **zero matches**. Chatterfly-named Container/Group Container roots and system Support/Caches roots were also checked for additional search roots.

This is a bounded negative for those roots and names. It does not exclude renamed, embedded, compressed-cache or memory-only data, unsearched roots or later provisioning. A runtime open-file snapshot likewise exposed no matching XNet/SentencePiece model path; that snapshot cannot establish memory absence after file handles are closed.

No model payload was available for format-aware architecture/weight analysis. Packaged opaque LM-named resources remain authenticated bytes with unknown full schemas. These limits remain open rather than being filled with a guessed checkpoint.

## Cloud configuration and routing

The selected allowlisted technical key in the installed `ChatterflyPY/UserPreferences.plist` is **`keyEnableCloudRequest = "1"`** (`CF-installed-pref-cloud`). The native wrapper reads that key via `SGInputConfig`'s `intObjectForKey:` and tests nonzero. The shipped brand defaults independently contain the same value. Only explicitly selected technical keys were retained; no personal preference fields or typing history were exported.

This answers the stored-preference question. Effective kernel state, network success and candidate provenance remain separate questions.

Cloud delivery has two paths. The kernel setting used by `currentCloudSendType` defaults to numeric 1, the LinkB/kernel path. Numeric 0 uses the LinkA/platform path; a circuit breaker can also route to LinkA. Consequently, a false return from the platform's `needRequestCloudCandidate` is not sufficient evidence that all cloud candidates are disabled. That kernel-setting index and the model `CoreConfig` key table are different enum spaces.

The earlier platform request path also checks input/candidate state, cache state, request arbitration, network availability, preference enablement and other mode conditions. The full LinkB scheduler, actual requests, server prompts and server model remain unresolved. No live request bodies or TLS traffic were captured.

## Actual keyboard probes

A read-only Carbon TIS query returned `com.tencent.inputmethod.chatterfly.pinyin` as enabled, selectable and selected. The user placed the test document in Chinese mode. Subsequent synthetic keys were sent through the computer-use tool to a separate TextEdit document, using individual key presses rather than pasting recognized text.

| Probe | Returned host AX state | Evidence |
| --- | --- | --- |
| `n i h a o` | `ni'hao` composition; separate Chatterfly-owned candidate window visible in window metadata | `CF-installed-ui-composition`, `CF-installed-candidate-window` |
| Continuous `n i h a o Space` in a cleared field | **你好** | `CF-installed-ui-commit` |
| `n i h a o Escape` in a cleared field | Empty field | `CF-installed-ui-cancel` |
| `n i h a o Backspace o Space` in an empty field | **你好** | `CF-installed-ui-edit` |

The candidate window belonged to the actual input-method PID, with a 558×60 visible rectangle during composition. These probes demonstrate real composition, cancellation, buffer editing and Chinese text delivery in one host. They do not trace internal model functions, attribute the winning candidate to local/cloud computation, test partial selection or establish broad accuracy. Synthetic results were retained locally under `.workspace/`.

These UI records transcribe actual computer-use tool observations; they are not REA Evidence IDs. Interrupted early probes produced Latin text during source/focus changes and are excluded from the Chinese-commit successes. Keeping input and selection in one continuous tool call yielded the verified Chinese result.

## Instrumentation limits and remaining work

REA native UI observation was attempted for the exact candidate PID/window. It returned `capability_unavailable`: the background input method exposes no application launch time, so REA could not perform its stable PID-reuse check. The failed request was retained, not retried, and supplied no canonical screenshot/accessibility Evidence. The ordinary host observations and target-window metadata remain separate evidence.

The executable is hardened and lacks `get-task-allow`, required by the documented REA LLDB call-observation workflow. No duplicate input-method process was launched under a debugger and no signing/security settings were changed. Native inference execution therefore remains untraced.

| Original gap | Follow-up status |
| --- | --- |
| Does the installed artifact match the extracted package? | Answered; common bytes match |
| Which local neural runtime/model roles exist? | Partially answered; concrete XNet LM and separate feature-sort/generation loaders |
| How do model enablement and defaults work? | Answered for inspected static predicates; active assignment unresolved |
| Are the named models provisioned on this Mac? | No matching assets in the bounded scan; deployment elsewhere remains unresolved |
| Is the cloud-candidate preference enabled? | Answered for stored preference; effective routing/traffic unresolved |
| Does actual keyboard composition/edit/cancel/commit work? | Observed for the listed synthetic TextEdit probes |
| Which neural functions ran for those probes? | Unresolved; no native inference trace |
| Offline behavior, measured latency, quality and server model? | Unresolved; no network-isolated test, benchmark or server evidence |

This follow-up adds evidence without changing Arion's product scope or selecting a keyboard engine/model for implementation.
