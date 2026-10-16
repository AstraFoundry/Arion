# Doubao and Chatterfly keyboard input

## Scope and evidence

This investigation follows physical-keyboard typing on the retained macOS packages: installed Doubao IME **1.0.2 (1)** and installer-extracted Chatterfly **1.0.6.13398 (13398)**. It covers input-method registration, keyboard events, Pinyin conversion, candidate ranking, composition, candidate UI and commit. It does not characterize newer builds, Windows or mobile implementations.

The [evidence ledger](evidence/keyboard-ime.json) separates actual REA Evidence IDs from ordinary package, metadata and byte-verified arm64 instruction observations. Matching existing REA Mach-O identities were reused; new REA `inspect_plist` records confirm input-host registration. Focused instruction analysis uses Apple LLVM and native symbols; it is not newly recovered REA pseudocode. The apps were not launched and no user dictionaries, account data or live traffic were read. Full local records remain in ignored `.workspace/research/keyboard-ime/`.

Both products contain native keyboard-conversion engines with conditional model/cloud enhancement. Their keyboard engines and the speech-recognition/post-processing paths described in [Doubao](doubao.md) and [Chatterfly](chatterfly.md) are distinct implementation surfaces. The exact enabled branches, server models and perceived typing latency require runtime evidence.

## System integration and composition

Doubao's plist registers `DoubaoImeInputController` and `DoubaoIme_Connection`; Chatterfly registers `YuanbaoInputController` and `Yuanbao_InputMethod_Connection`. Both are background input-method bundles. These are system input sources hosted through InputMethodKit.

Apple's [InputMethodKit documentation](https://developer.apple.com/documentation/inputmethodkit) describes communication between input methods and client applications. An [IMKInputController](https://developer.apple.com/documentation/inputmethodkit/imkinputcontroller) manages input-session events and the engine's converted text. The installed SDK documents event delivery through `handleEvent:client:` or the `inputText` variants, marked-text composition and final insertion. The framework supplies the host protocol; each product supplies its conversion engine, candidate policy and UI.

The important states are:

```text
physical key event
  -> input-method controller
  -> editable input buffer and conversion state
  -> local Pinyin decoding and candidate ranking
       + conditional cloud/model candidate updates
  -> marked text in the client + candidate window
  -> candidate selection / explicit commit
  -> insertText into the client and clear the composition
```

This diagram is a bounded reconstruction from the functions below. It is not a recorded execution trace. Marked text remains provisional; final insertion completes the current conversion. Engine state can retain an unconverted suffix after selecting a partial candidate.

## Doubao

### Controller and engine boundary

The main executable's relative Objective-C metadata identifies `DoubaoImeInputController` as an `IMKInputController` subclass, including `handleEvent:client:`, `recognizedEvents:`, `commitComposition:`, activation and deactivation methods. The proprietary `OimeEngine.framework` exposes an Objective-C/Swift wrapper over a symbol-rich C++ engine (`DBK-host`, `DBK-local-engine`).

Relevant engine operations include conversion, candidate access, selection, commit, composition and caret movement. This is an independently stateful keyboard engine. Installing an IMK bundle alone does not provide its Pinyin conversion behavior.

### Local conversion and ranking

Byte-verified instructions in `oime_engine::pinyin::Engine::PinYinConvert` call the syllable-lattice builder, user-dictionary and system-dictionary processing, and sentence conversion. `CanditateSortByFeature` calls `MLPModel::CalcScoreByFeature` (`DBK-asm-pinyin-convert`, `DBK-ranking`). The recovered components therefore support this reconstruction:

```text
input letters / Pinyin scheme
  -> syllable lattice
  -> system words + user words
  -> sentence candidates
  -> feature extraction and local MLP scoring
  -> ranked candidates
```

Symbols also identify bigram and user-dictionary machinery. Their presence supports local lexical/contextual ranking capabilities; it does not reveal all scoring features, training data, learned weights or every active branch. The MLP ranker is a concrete keyboard-ranking call site. The separate ASR transformer/kernel names discussed in the speech report do not identify this ranker's model.

### Cloud and LLM enhancement

The engine wrapper includes `makeCloudRequest`, `makeLLMRequest`, `parseCloudResponse:llmResponse:precedingText:onTiming:sessionLLMEnabled:`, `updateLLMResult` and `getLLMCandidates`. A focused network wrapper contains the actual selector call `sendLLMRequestWithInput:textBeforeCursor:textAfterCursor:type:scheme:timeout:enableLogger:callback:` (`DBK-cloud`). This establishes an additional request/response and candidate-fusion surface that accepts input, surrounding text and keyboard scheme.

The request wrapper does not establish which requests the installed configuration sends, the server checkpoint, its prompt or whether every keystroke triggers a request. The current [vendor page](https://shurufa.doubao.com/pc) separately advertises model-assisted association, long-sentence completion and correction. That is a vendor capability claim, not a measurement of this retained build.

### Candidate UI and commit

The package exposes its own candidate/controller and composition state. The main controller's commit method and the engine's candidate-commit functions provide the host/engine endpoints (`DBK-host`, `DBK-commit`). The complete Swift keyboard-selection-to-host-insertion call chain is only partially reconstructed; speech paste-gate decompilation from the earlier report must not be relabeled as the keyboard commit path.

## Chatterfly

### Keyboard dispatch and conversion

The retained executable preserves Objective-C and C++ names. Focused instructions connect `SGBaseInputController` event handling, `YuanbaoInputController` key handling and `SGInputControllerPy` to `SGPyConversionEngine`. Its `addInput` mutates the composition input strings and invokes `convertWithHint:` (`CF-key-base-event`, `CF-key-yuanbao-key`, `CF-key-py-key`, `CF-key-engine-add`).

The conversion bridge calls `SGConversionEngine`, `SGCoreService` and `pc_convert_any_to_array`. Deeper native functions identify `_sgime_core_pinyin_::CSogouCoreEngine` and the `sogou_ime_kernel` family. Actual conversion instructions enter Pinyin-network parsing, syllable-list construction and `SentenceProcess_DP` (`CF-key-bridge-convert`, `CF-key-core-convert`, `CF-key-pc-adapter`, `CF-key-manager-convert`, `CF-key-py-convert`). These observations establish substantial Sogou-named keyboard-engine lineage in this build; they do not establish source-code identity with a particular public Sogou release.

### Ranking and packaged data

The recovered decoder uses a dynamic-programming sentence path, followed by candidate construction and ranking surfaces including `FillWord`, `SmartFuzzy`, `SortCandsByLtr` and language-model feature sorting. The inspected sorting function builds context/dictionary/language-model features and calls `LMInferWithCloudCache`. Its scoring implementation checks `LocalLanguageModel::IsAttach` and directly calls `LocalLanguageModel::GetScore` (`CF-key-py-result`, `CF-key-lm-sort`, `CF-key-lm-infer-impl`). This is a concrete local language-model scoring capability; its exact architecture and enabled settings remain unresolved.

A conditional branch also calls `StartGenerateModelInferenceAsync`. That function checks enablement, trigger conditions and `IsNeedGenerateCand` before running generation. A loader references `xnet_enc_model.xnet`, `xnet_dec_model.xnet`, `sentencepiece.model` and `sentencepiece_dec.model`; those exact filenames are absent from the inspected retained bundle. Provisioning outside that bundle and execution remain unknown (`CF-key-generate-async`, `CF-key-generate-loader`, `CF-key-generate-infer`). It is therefore accurate to report a conditional generative-candidate implementation, without assigning it a checkpoint or assuming it runs on each key.

`Contents/Resources/ImeKernel.bundle` contains the following observed files (the ledger's `chatterfly.resources` inventory):

| Resource | Size | Interpretation limit |
| --- | ---: | --- |
| `ime_sys.bin` | 4,438,030 bytes | System-dictionary filename and retained bytes; complete format not decoded |
| `ime_gd_bigram.bin` | 10,777,984 bytes | Bigram-named data; exact scoring semantics unresolved |
| `ime_lm.bin` | 532,485 bytes | Language-model-named data; architecture not established from its name |
| `ime_em.bin` | 79,776 bytes | Opaque data; filename alone does not establish the model |
| Correction/fuzzy/Pinyin configuration | Multiple files | Explicit configurable keyboard conversion surfaces |
| `ShuangPinSchemes/` | Multiple scheme files | Packaged double-Pinyin mappings |

The specific XGBoost-named loader references a name-model resource; this must not be generalized into a claim that all sentence/candidate ranking uses XGBoost (`CF-key-xgb-load`). Likewise, language-model symbols do not identify a public generative checkpoint.

### Local results, cloud candidates and UI

Within `convertWithHint:`, local conversion calls precede cloud-candidate preparation and the conditional `startRequestCloudCandidatesWithBundleID:` call (`CF-key-engine-convert`). This is a local conversion stage with an additional cloud-candidate path. No network dependency or offline accuracy was measured.

`SGCandidatesController` updates client marked text with `setMarkedText:selectionRange:replacementRange:` and then calls `updateWindow:`. Candidate presentation is managed by a custom native controller/window. It is not established merely by finding the standard `IMKCandidates` API (`CF-key-ui-update`, `CF-key-ui-window`).

Candidate selection is stateful: `selectCandidateAtIndex:` can consume part of the input and reconvert the remainder. The ordinary text-commit path reaches `SGInputHelper` and `sg_insertText:sender:replacementRange:shouldRecord:`, which explicitly calls the sender's `insertText:replacementRange:` (`CF-key-engine-select`, `CF-key-helper-commit`, `CF-key-helper-insert`). This establishes a keyboard commit path into the host application, independently of speech delivery.

The selection path also calls `learnWord`, which passes candidate/Pinyin metadata into delayed learning and later `learnWordCommit` (`CF-key-engine-learn`, `CF-key-engine-learn-submit`). This establishes a learning path; actual persisted personal data and long-term adaptation were not inspected.

## Comparison and implications for Arion

| Responsibility | Doubao | Chatterfly |
| --- | --- | --- |
| macOS integration | IMK controller and registered input source | IMK controller and registered input source |
| Local keyboard core | `OimeEngine`, C++ Pinyin engine | `SGPyConversionEngine`, Sogou-named C++ core |
| Pinyin decoding | Syllable lattice, system/user dictionaries, sentence conversion | Pinyin network, syllable list, dynamic-programming sentence processing |
| Ranking evidence | Local MLP feature-scoring call | Context/dictionary/LM features, LTR sorting and model-inference calls |
| Enhanced candidates | Cloud/LLM request, parse and fusion interfaces | Conditional cloud-candidate request and conditional generation-model inference |
| Candidate UI / final delivery | Custom state and commit endpoints; complete keyboard host chain partly unresolved | Custom candidate window, marked-text updates and explicit host `insertText` chain |

For Arion, adding full keyboard Pinyin would require a separate keyboard-engine capability: scheme normalization, incremental parsing, lexicon and sentence search, ranking, user-word learning, partial selection, editing and candidate pagination, plus IMK/TSF adapters. The existing voice IME host covers only part of that work. Reusing an appropriately licensed keyboard engine is an implementation option, not a conclusion about either competitor's source code.

This research does not change Arion's approved scope. [The product boundary](../project/README.md#explicit-boundaries) and [specifications](../../specs/README.md) keep a full conventional Pinyin keyboard behind a separate scope decision.

## Coverage checklist

| Question | Status |
| --- | --- |
| Are these native system input methods? | Answered for both retained macOS packages |
| Is there a local keyboard conversion engine? | Answered by instructions and packaged symbols for both |
| How are Pinyin and candidates processed? | Partially answered; concrete decoding/ranking call sites recovered |
| How are candidates displayed and committed? | Chatterfly ordinary text path answered; Doubao keyboard host chain partially answered |
| Do cloud/model keyboard branches exist? | Answered for shipped capability; runtime selection unresolved |
| Exact algorithms, model weights, server prompts and quality? | Partially answered for local algorithm families; remaining details unresolved |
| Actual offline behavior and end-to-end latency? | Unresolved; no runtime experiment was run |

Reproduction and exact artifact/function hashes are in the ledger. No persistent native disassembler session was opened during this pass.
