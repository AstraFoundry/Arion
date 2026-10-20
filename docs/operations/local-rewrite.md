# Local rewrite product configuration

The macOS Apple Silicon product selects `qwen3.5-2b-q4-k-m-metal` from [the model catalog](../../models/catalog.json). The user authorized adoption of the current baseline, accepting its known three harmful choices and six missed corrections in the exposed 101-case selector development set. These are local spelling-choice observations, including 12 mechanically forced-original cases, not a complete-product semantic success rate or a 16GB host qualification.

## Model and runtime

- Checkpoint: Qwen3.5-2B, actual 1,881,825,088 parameters; pinned community `Q4_K_M` GGUF, 1,280,835,840 bytes, SHA-256 `aaf42c8b7c3cab2bf3d69c355048d4a0ee9973d48f16c731c0520ee914699223`.
- Export: `unsloth/Qwen3.5-2B-GGUF@f6d5376be1edb4d416d56da11e5397a961aca8ae`; source `Qwen/Qwen3.5-2B@15852e8c16360a2fea060d615a32b45270f8a8fc`; Apache-2.0. Source/export numerical parity is not established.
- Runtime: llama.cpp `b11734`, source `f3fb2fcdbc93a8f677e893b8dbd36462b3190ffd`, MIT. [Runtime pins](../../scripts/llm-runtime.json) verify the archive before staging its executable, dependency closure and notices. GenAI libraries are no longer staged for rewriting; speech still uses its independent ONNX runtime.
- One Metal slot, 4096 context tokens, 4096 batch/microbatch, four default CPU threads, seed 42, greedy decoding, thinking disabled. Selector output budget 128; formatter budget 256. Complete native template/token count is checked before inference; over-budget context fails explicitly without truncation.
- `arion-llm` owns the native child through a private Unix socket in a mode-0700 temporary directory. No TCP listener, proxy, remote inference or automatic model substitution is used. Runtime logs are discarded. Slot erasure requires the pinned runtime's `--slot-save-path` capability; no save/restore request is made and no KV file is written. The slot is erased at session boundaries; its directory is removed on close. The client owns a separate process group and kills surviving descendants if the worker exits.
- The preparation operation renders/tokenizes and checks budgets without sampling. It does not precompute native KV state: zero-token chat completions in the pinned runtime still sample a token. No generation or prefill speedup is claimed for preparation.

Weights remain outside Git, under the profile's `models/<id>/` directory. Existing provisioning verifies every file's size and SHA-256 and atomically publishes the model. The old ONNX rewriter is absent from the active catalog; there is no silent fallback. Intel and Windows remain unsupported for local rewriting.

## Prompt ownership and delivery

The shipped identifier is `selector-screen-v1-baseline+natural-layout-v1`:

1. `domain.CleanSpeech` performs the existing local deletion stage.
2. `domain.SpellingLexicon` derives maximal nonoverlapping source-relative alternatives using the pinned pypinyin character readings and CMU English dictionary. Chinese matching is tone-insensitive; English matching uses 2..6-word spans and prioritizes identical letters with canonical spacing/case. Quotes, numbers and exact Chinese spellings are excluded. These are unverified hypotheses; ambiguity can only keep the source.
3. [Selection template](../../apps/desktop/infrastructure/llm/prompt/selection.json) uses the exact evaluated baseline system and ten examples. Full sanitized OCR and available cursor/history/approved terms remain reference data. Only the complete keep/replace ID protocol and exact UTF-8 spans are checked.
4. [Formatting template](../../apps/desktop/infrastructure/llm/prompt/format.json) uses the natural-layout-v1 recipe. It receives only the edited source, routes examples/system rules by its actual Han/Latin script profile, and retains separate work, email, coding-prompt and chat scenes. Search uses chat plus existing mechanical punctuation handling; explicit list has its own numbered-list recipe. Scene hints do not imply list intent: work/prompt default to minimal copyediting of natural prose and questions; only clear enumerations or explicit list requests permit new list formatting. Existing layout, sentence order and voice are preserved, and already usable text may remain unchanged. No new headings or category labels are requested. OCR/context never reaches this stage.
5. `ValidateRewriteDelivery` checks complete nonempty UTF-8 output and control markup. It does not reject on overlap, wording changes, added numbers, negation counts or OCR evidence. `ValidateRewrite` remains a historical diagnostic for benchmark replay, not a production semantic veto. This permits model mistakes; natural layout is a generation policy and a review objective, not a semantic guarantee.
6. Cancellation, session/source revision, focus, mode and single-commit ownership checks remain enforced. Transport failure, timeout, incomplete output or unavailable model is explicit; the existing completed-cleanup delivery behavior remains. The recognized original is retained separately.

The historical 457-case producer parity fixture and 101 selector / 16 formatter payload parity observations are under [product integration reproduction](../../benchmarks/reproduction/rewrite-product-integration-v1/). They check the original packaging/porting equivalence, not fresh quality or the revised formatter. Current tests retain selector byte/payload parity and verify formatter source/context isolation and routing separately. Natural-layout model observations are tracked in [the focused verification](../../benchmarks/reproduction/rewrite-natural-layout-v1/README.md). Actual worker observations and known limits belong in its integration report. Current development validation uses M4/32GB; Apple Silicon/16GB, other supported macOS versions, concurrent audio and target-editor acceptance remain separate hardware/product checks. The user has no 16GB host available and authorized continuing on the development host.

## Local verification

```sh
go test ./apps/desktop/infrastructure/llm/... ./apps/desktop/infrastructure/rewriting ./apps/desktop/infrastructure/provision ./models/... ./packages/core/...
make build desktop
```

The build uses the shared resource staging and verifies packaged architectures, minimum OS metadata, licenses and signatures. Do not put weights in the application bundle or commit them with templates. Retired research weights may be removed while retaining original raw outputs, hashes, provenance, failed attempts and reproduction sources; rerunning those retired models then requires downloading their pinned artifacts again.
