# Ordinary-word meaning, screen context and pronunciation examples

This probe investigates the reported `codex能爆吗 → codex能报吗`, followed by `codex能播吗` in a reimbursement conversation. It uses **actual Qwen3-1.7B text generation**, production edit extraction/encrypted storage, Go prompt rendering and the product validator. Inputs are reported/authored ASR strings and reconstructed screen text. It does not recognize audio, capture OCR or inspect the user's actual application.

## Established structural causes

The product turns screen text into `domain.ContextTerms`: sender/quoted names, Latin names, identifiers and codes. Ordinary conversational meaning can be lost. The rewriter receives these terms, not the complete OCR excerpt; the bounded excerpt still reaches validation. A benchmark-only section template can send complete context, but primarily instructs name/identifier spelling. [Code and paper analysis](../../../docs/research/closed-loop-memory.md) separates those facts from unverified claims about the reported run.

The current data model already permits multiple source forms pointing to the same destination. Exact matching remains a limitation: `播吗` does not match stored `爆吗`. The experimental pinyin retriever does find the stored pair at distance 0.2. Both `报` and `爆` have `bao4`; `播` has `bo1`. Pinyin connects confusions but cannot resolve identical pronunciations by itself.

The production extractor stores `爆吗 → 报吗` in the development probe. Other examples expose fragment selection: `公示需要 → 公式需要` yields `示需 → 式需`; `全免的 → 全棉的` yields `免的 → 棉的`. Widening a changed CJK character with its neighbor is not word segmentation. A different following word can prevent subsequent retrieval even when the intended lexical correction is the same.

## Frozen comparison

Three development semantic families produce six competing-meaning branches and six preservation controls. The reported Codex case is development only. Six disjoint test families produce ten meaning branches and six controls. Prior correction pairs, screen text, inputs, references, generic instructions, selection rule and output ceilings are frozen before any generation. Test references never enter prompt construction or memory.

| Condition | Mechanism |
| --- | --- |
| `product` | Current global vocabulary application, exact relevant pairs and extracted context terms; actual shipped prompt |
| `sections` | Raw input, retrieved stored arrow pairs and complete bounded screen text; existing section prompt |
| `semantic_context` | Same raw input/context/retrieval; generic instruction allowing local homophone disambiguation without changing the speaker's facts or answering their question |
| `phonetic_context` | Same semantic instruction/context; actual stored corrections serialized as observed form, corrected form, text-derived pinyin and query distance in one user turn |

The `product` contrast changes pre-application, retrieval and context delivery together, so it cannot isolate each change. `sections` versus `semantic_context` tests the instruction change; semantic versus phonetic compares retrieved-example representation. Latin strings are not converted to English phonemes. None of these is acoustic few-shot learning, a trained phoneme model or a G-SPIN reproduction.

The earlier user feedback is passed through production `ExtractCorrections` and the encrypted store from an empty vault; every returned snapshot is reopened. Subsequent text cases cannot mutate that memory. Full prompts, actual generations, tokenizer parity, raw text, validator outcomes and delivered text remain in [the runs](../../runs/context-memory-v1/).

## Development decision

| Condition | Delivered exact / 12 | Raw model exact / 12 | Preservation controls failed / 6 | Originally correct inputs regressed / 9 | Validator rejections |
| --- | ---: | ---: | ---: | ---: | ---: |
| Product | 8 | 8 | 1 | 2 | 0 |
| Existing sections | 8 | 7 | 1 | 1 | 1 |
| Semantic instruction | 8 | 4 | 1 | 1 | 4 |
| Retrieved pinyin examples | 9 | 4 | 0 | 0 | 5 |

The predeclared guards select the pinyin-example candidate for one final test. All four conditions match **3/6 meaning branches**: its delivered-score gain comes from avoiding a preservation failure, not improved disambiguation. Five invalid generations are rejected and the original source is delivered; those retained correct inputs must not be credited as successful model corrections.

All conditions leave the reimbursement branch as `codex能播吗？`, including full screen context plus the retrieved `爆吗 → 报吗` example. The phrase is therefore **not fixed by this probe**. Product global application also changes the unrelated literal question `气球会爆吗？` to `气球会报吗？`; validation against the already modified source does not detect the original meaning change.

## Final test and rejection of the candidate

| Condition | Delivered exact / 16 | Raw model exact / 16 | Meaning branches / 10 | Preservation controls failed / 6 | Correct inputs regressed / 8 | Validator rejections |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Product | **10** | **10** | **4** | 0 | 2 | 0 |
| Existing sections | 9 | 8 | 4 | 1 | 1 | 1 |
| Retrieved pinyin examples | **8** | **3** | **2** | **0** | **0** | **7** |

The candidate does not generalize into a quality gain: it preserves correct inputs but corrects fewer ambiguous ones. Seven model outputs are rejected; five contribute a correct retained input, explaining the gap between 3 raw and 8 delivered exact matches. The product condition also has one question-preservation violation; the other conditions have none. No arm reaches the token ceiling, and the explicit number/negation/name proxy has no violation. These proxies are not independent semantic ratings.

There is no new adoption and no prompt revision after this test. Development and test each contain 48 actual text-model generations plus one warmup: **96 measured generations and two additional warmups**. CPU generation with repeated full-prefix prefill has final-test p50/p95 of 7.70/10.03 s (product), 13.50/17.61 s (sections) and 10.00/12.67 s (pinyin examples). This is not native cached-prefix product latency.

[Test summary and every scored result](test-summary.json), [development summary](dev-summary.json), [the frozen selection](../../runs/context-memory-v1/selection.json), [exact fixtures](../../corpora/context-memory-v1/), [reproduction instructions](../../reproduction/context-memory-v1/README.md) and [archive identities](archive.json) preserve the decision sequence. The negative result applies to these specific prompt/representation candidates and checkpoint; it does not establish that all phonetic retrieval or short constrained-choice approaches fail.

## Limits

References are single-author exact-string proxies, not independent intent review. Screen text is reconstructed, not actual OCR; reported ASR strings are not recordings. Different context branches of the same text are paired counterfactual diagnostics, not independent users. Some branches have linguistic ambiguity. The ordinary-word text probe and separate [complete-audio learning loop](../user-modify-asr-v3/README.md) do not together establish a measured native product pipeline.

The text checkpoint is the pinned community CPU int4 Qwen3-1.7B export from the historical diagnostic study, with ONNX GenAI 0.17.1 / ONNX Runtime 1.30.0 and four CPU threads on Apple M4/macOS arm64, 32 GiB. It is different from Qwen3-ASR-1.7B. Model bytes/tokenizer identities and the derived CPU configuration are verified; source BF16 parity is unmeasured. Weights, runtime environments, helper binaries and vault keys remain private.

The Python worker regenerates the whole prompt prefix for every request with resident weights. Its generation compute includes repeated prefill and does not measure the native product's prefix reuse or stop-to-insertion latency. Rejections and truncations remain explicit. No product prompt, context behavior or memory schema is promoted by this diagnostic.

```sh
pnpm memory:context-report benchmarks/corpora/context-memory-v1 \
  benchmarks/runs/context-memory-v1/test-run \
  .workspace/context-memory-v1-test-replayed.json
```
