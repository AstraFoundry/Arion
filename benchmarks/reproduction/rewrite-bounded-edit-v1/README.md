# Bounded spelling edits and isolated formatting

Research package only. Qwen3.5-2B Q4, actual1.882B text tensor elements, native arm64 llama.cpp b11734, CPU4/GPU0/cache off. No new4B, older Qwen, Granite or Q8 inference. [Frozen protocol](protocol.json) predates all72 requests; [freeze](freeze.json) binds producer, prompts, input and resource bytes. The fixed pypinyin0.55.0 table contains all41,923 built-in characters, not test-derived names; its MIT source/license and CMU provenance are retained.

All36 exposed Chinese synthetic regressions receive an actual selection call (128-token cap) and a formatter call (256). Full original OCR/context goes to selection. Grammar permits only source-relative spelling decisions; ambiguous candidates can only keep their source in this version. The formatter receives edited source plus opaque token types, without OCR or restoration spellings. It freely rewrites prose and layout. Unfiltered restoration never rejects an output or substitutes source. This differs from production CTX-05 routing and has not been adopted.

All72 raw outputs, hypotheses, edits and private mappings were personally inspected. Intermediate33/36 decisions conform,8/11 necessary repairs. Final bodies28/36 faithful,27/36 scene-ready,6/11 repairs. Three necessary repairs are kept; formatting drops two corrected recipients, changes no-config-change to later-change-config, loses a named target and changes negative-question polarity. Owner/deadline bullets and leave-email paragraphs succeed. [Selection](../../runs/rewrite-bounded-edit-v1/run/selection.json) is unqualified; fresh24 remain unused, production unchanged. Forced empty/ambiguous JSON choices do not establish model semantic robustness.

The later [user-confirmed spelling policy](../rewrite-spelling-alignment-v1/README.md) allows read cash→readCash, while still allowing raw spelling. Thus this version's conservative keeps do not themselves fail that clarification. Necessary typo repairs, dropped names and semantic regressions are separate failures. Neither new nor historical original scores are silently rewritten.

```sh
pnpm rewrite:bounded:report --out .workspace/bounded-report-repeat
pnpm rewrite:bounded:test
pnpm rewrite:bounded:verify
```

Report/verify load no model. New inference requires the exact separately provisioned model/runtime from freeze and a new output directory:

```sh
pnpm rewrite:bounded:run --out .workspace/bounded-new-run
```

All72 original rows/logs/props and per-row judgments are in [run](../../runs/rewrite-bounded-edit-v1/run); report summary and complete messages/maps are replayable. Actual production-cleanup snapshots retain original source spans independently. Inference received no evaluator expectations or intended text. M4/32GB development, sampled worker/coordinator RSS, serial requests and two calls are not whole-app peak memory, minimum16GB acceptance or concurrent stop-to-insertion latency. Independent natural ASR/OCR, English/mixed, search/list/verbatim and native lifecycle gates remain.

A separate [producer audit](../../runs/rewrite-bounded-edit-v1/producer-language-audit.json) runs no LLM. The English matcher finds a Chinese-surrounded build cache span but misses the same span embedded in longer English/mixed phrases. This is a producer coverage gap; it cannot be fixed by a formatter prompt. Original frozen producer is preserved for replay.
