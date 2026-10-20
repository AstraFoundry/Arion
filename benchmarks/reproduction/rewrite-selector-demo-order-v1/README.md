# Existing demonstration order only

This study moves the original three keep demonstrations to the end of the same ten complete user/assistant pairs. The zero-based permutation is `[0,1,2,3,7,8,9,4,5,6]`. No demonstration, answer, rule or evaluated source is added, removed or edited. The complete system message, current source/fullOCR, hypotheses, status/kind, schema and decoding settings stay unchanged. Inverting the permutation restores all202 treatment requests exactly to their current controls. This tests this specific ordering; it does not establish a general causal account of recency bias.

Eight payload checks cover all202 paired requests and exact inverse restoration. Native b11734/Metal99 performs404 template and404 tokenization requests with zero model generations. Maximum complete input1891 plus128 reserved output tokens fits4096. All202 original control token counts and all404 later actual inference input-token counts match. Full rendered prompts, token IDs, commands, properties and source references are retained without shortening the OCR.

The770-file freeze precedes404 actual choices. Actual1,881,825,088-element Qwen3.5-2B Q4 uses b11734/Metal99/CPU4/context4096/parallel1/greedy42/reasoningoff/cachetrue/128tokens/no-warmup. All16 actual scene workers offload25/25 layers.457 fresh actual Go preparations are reused across four arms;1828 source-treatment outcomes include1424 explicit no-hypothesis skips. No new ASR, Vision, formatter or final-body inference occurs. There is no new4B or olderQwen model.

| Arm | Conforms /101 | Harmful cases | Missed repair cases | Repairs /47 |
| --- | ---: | ---: | ---: | ---: |
| Baseline |92|3|6|41|
| Keep examples last |78|2|21|26|
| Balanced |94|4|3|44|
| Keep examples last + balanced |88|2|11|36|

All404 actual raw choices, edited sources and full source/context/hypothesis tuples were personally read. All25 unique failed tuples and all27 changed within-run pairs were separately reviewed. Three changes improve,23 regress, and one remains allowed under the same-letter normalization rule.202 current controls reproduce prior raw choices and edited sources exactly. All responses stop normally with zero protocol failures. Each arm has12 original-only ambiguity schemas; their success is not autonomous semantic judgment.

## Decision

Reject both reordered configurations. Reducing harmful substitutions by one or two cases loses15 or8 required repairs. Explicit names such as 吴雨辰 with OCR 吴语宸, and document terms such as 指保书 with OCR 质保书, remain unrepaired. The balanced treatment still substitutes 按时 with unrelated 按实 and customer-interaction 大理 with unrelated 大里. Moving keep examples to the end produces a different error balance without solving contextual grounding.

Keeping `get cash` or normalizing to `getCash` is allowed; unsupported `readCache` is not. These scores do not penalize structural rewriting: no final bodies are generated in this study. Existing ID/enum/UTF8 span application remains, with no semantic output rejection or source fallback. No configuration qualifies for expansion or production adoption.

The frozen original corruption test selects a nonexistent arm name and raises StopIteration before testing changed context. Its source and failed attempt are retained in `test_report.py` and the run's `original-test-failure.json`. [Version2](test_report_v2.py) selects the actual negative-last arm; all11 evidence checks remain. This correction changes no inference, result, score or original freeze. The eight pre-inference payload checks passed independently.

Observed serial request medians are267/278/538/493ms; p95 is399/430/1139/1315ms. These exclude startup, Go preparation, formatting, native insertion and concurrent ASR, and do not establish a product SLO. Sources are exposed development material, not held-out or independent blind evaluation. Full-chain/fresh/currentASR+Vision/cross-script/native lifecycle/offline/concurrent/independent review and actual Apple Silicon16GB acceptance remain open. The user currently has no16GB Mac and authorized continuing32GB development-host validation; the minimum target is unchanged.

## Reproduction

Use `pnpm rewrite:selector-demo-order-v1:run --out <new-private-directory> --cleanup-worker <pinned-worker>`; `:payload-test`; `:test`; `:verify`; `:report --out <new-directory>`. The test alias executes the documented version2. The unchanged executor's historical selector-screen-v1 labels remain; model/freeze/actual requests identify this treatment. New prompt identity points to owned layout and pinned original system/example sources. Weights, runtimes and native worker binaries remain private.
