# Technical spelling formatter verification

Natural-layout-v3 permits established English technical terms expressed phonetically in Chinese to regain English spelling in unambiguous technical context. It preserves surrounding Chinese, named companies/projects, literal quotations and explicit Chinese naming. The selector and pronunciation producer remain unchanged; this is LLM formatting rather than dictionary-wide replacement.

`cases.json` contains ten assistant-authored development text probes. `TestNativeTechnicalSpellings` uses the same actual-worker harness as numeric verification, with live and final flags and the existing 10-second request deadline. The model never receives the expected-output checks. Recorded worker outputs are not acoustic recognition, full streaming session, editor insertion, blind review or held-out quality evidence.

Baseline and revised outputs, failed attempts, sources, provenance and hashes are retained here. Baseline passes 10/16 declared checks over the original eight sources. The first policy revision passes 12/16: alpha improves, beta release and delta still fail. The accepted revision adds parameter/algorithm and release demonstrations and passes 20/20 requests over ten sources; the last two are added preservation probes (implicit company name and ambiguous fragment). All accepted live/final pairs match. The assistant reviewed the outputs for term spelling, protected names, negation and constraints; this is not an independent quality estimate.

`summary.json` reports these development checks. `SHA256SUMS` covers the retained files; run `shasum -a256 -c SHA256SUMS` from this directory to verify original bytes without inference. Runtime/model pins, host/power, decoding settings and request budget are in `provenance.json`. See [product configuration](../../../docs/operations/local-rewrite.md). Reproduce with the pinned runtime/model and a fresh output path using the numeric verification command but with `-run TestNativeTechnicalSpellings`.

The accepted formatter also passes all 24 numeric/hesitation live/final regression requests under the 10-second deadline. Raw responses remain separate from the technical corpus; no ASR or stop-to-editor latency claim is made.

All 14 authored layout probes plus session reset, generation-free preparation, cancellation and recovery checks pass in the actual worker. Layout responses were compared with the prior numeric verification and reviewed for preserved facts, constraints and explicit-list behavior.
