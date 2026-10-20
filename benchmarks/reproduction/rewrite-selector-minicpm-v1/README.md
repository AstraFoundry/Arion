# MiniCPM5-1B native and unchanged selector comparison

Both configurations fail the unchanged development gate. All202 actual selector outputs,12 complete native probe responses,32 unique failed full-source/context/output tuples,55 changed pairs and1 interrupted native-log completion were personally inspected. Assistant review is not independent blind release acceptance.

| Configuration | Conforming /101 | Harmful | Missed repairs | Required items repaired /47 |
| --- | ---: | ---: | ---: | ---: |
| Prior actual Qwen baseline |92|3|6|41|
| MiniCPM baseline |80|11|10|37|
| Prior actual Qwen balanced |94|4|3|44|
| MiniCPM balanced |77|4|20|27|

The202 exact complete requests and prepared candidates match prior actual controls.55 choice changes comprise6 improvements,35 regressions and14 allowed normalization differences. The original ten examples, full OCR, hypotheses, schema,128 output-token limit and temperature0/seed42 remain unchanged. Twelve forced-original ambiguity schemas per arm are not autonomous semantic decisions. Allowed `read cash` → `readCash` and unchanged spelling receive credit; unsupported `readCache` does not.

The official posttrained [MiniCPM5-1B](https://huggingface.co/openbmb/MiniCPM5-1B) and [official GGUF](https://huggingface.co/openbmb/MiniCPM5-1B-GGUF) are pinned in `freeze.json`:1,080,632,832 actual GGUF elements,219 tensors, Q4_K_M,688,065,920 bytes, SHA256 `81b64d05a23b17b34c475f42b3e72fbde62d4b92cc34541f7a8031d0752deafa`, Apache2. The marketed2B sibling is excluded because metadata reports2,516,756,480 parameters. No4B or olderQwen inference. Model, architecture, tokenizer and template change together; this is not parameter-count-only causality.

Twelve native probes use Metal99 and CPU0 with pinned b11734,4 threads,4096 context, parallel1, `--reasoning off`, no warmup, explicit offline operation. Actual logs show25/25 and0/25 GPU layers, thinking0 and the empty thinking-block generation prefix. All12 stop with valid JSON; positive cache-off/on/off raw parity holds. All12 semantic probe judgments fail, both with and without schema. Runtime compatibility permits only the separately prefrozen comparison, never release adoption. CPU0 is an explicit diagnostic, not an automatic fallback; operation offload may still occur.

The pinned embedded template equals the pinned upstream standalone Jinja file. Initial qualification incorrectly expected native `/props` bytes to include the final LF. Pinned `common/jinja/lexer.cpp` explicitly removes one trailing newline; `common/chat.cpp` stores this lexer source. `comparison-addendum.json`, exact source bytes/receipts and native `/apply-template` output disclose this verification correction. No general whitespace tolerance or prompt change is introduced. The launcher accidentally continued after the first assertion failed; it was interrupted and the owned server terminated. Zero structured responses were captured, but one completed `{"e0":"keep"}` is retained in the native log and separately reviewed as a missed supported repair. Its original running manifest, interrupted terminal status and closure remain visible. It is not a fabricated response envelope or a complete case. Earlier weight-inspection failure is also retained in `sources/`.

The original378-file inference freeze remains immutable. The comparison uses an exclusive new directory after corrected native qualification; its terminal status binds the addendum SHA.457 actual historical Go preparations are reused across914 treatment outcomes, including712 explicit no-hypothesis skips. No new cleanup, ASR, Vision, formatting, final body or native target insertion. Legacy shared executor labels remain unchanged; owned freeze/model/commands identify the real experiment.

Reproduction from the repository root:

```sh
pnpm rewrite:selector-minicpm-v1:test
pnpm rewrite:selector-minicpm-v1:verify
pnpm rewrite:selector-minicpm-v1:report --out .workspace/minicpm-replay-new
```

Matching Make targets use hyphens and `ARGS`. Fresh inference requires the pinned private model/runtime and new output directories. `python3 benchmarks/reproduction/rewrite-selector-minicpm-v1/preflight.py --out <new-private-directory>` runs the fixed12 probes. Verify actual template via the exact lexer normalization, protocol, nonthinking prefix, layer placement and cache gate as recorded in the qualification sources before `pnpm rewrite:selector-minicpm-v1:run --out <another-new-directory>`. The shared executor does not automatically enforce that manual preflight gate. Do not overwrite archived results or regenerate prior controls for report replay.

Passing this component would only permit complete457-source chain and unchanged fresh/currentASR+Vision, cross-script, native cancellation/focus/offline, ASR concurrency, independent review and actual Apple Silicon16GB acceptance. None follows from this M4/32GB diagnostic. Both arms are rejected; production prompts, validators, provider and native integration remain unchanged.

Post-run [official/native tokenizer comparison](../rewrite-minicpm-tokenizer-v1/README.md) verifies identical token ID sequences for all202 complete native prompts, with zero added model generations. It does not establish weight/logits or semantic equivalence; the original rejection is unchanged.
