# Model catalog

`catalog.json` is the reviewed, embedded catalog ([006 Models](../specs/006-models.md), [ADR 004](../docs/decisions/004-automatic-model-selection.md)). Weights are never committed: the app downloads them from the pinned URLs and verifies them.

## Artifacts

Each artifact records its ID and purpose (`streaming-asr` or `offline-asr`), sherpa-onnx model kind and runtime version, the upstream archive URL, release, publication time, archive SHA-256 and byte size, model card and source repository, model and runtime license references with a review status, languages, platforms, audio format, and every loaded file: its archive member path, SHA-256 and size. Latin casing, punctuation, quantization and conversion provenance are explicit; vendor training notes are kept, including undisclosed test-set exposure.

Provisioning downloads into a bounded, resumable partial file, verifies the archive digest, extracts only the listed members into a staging directory, verifies each member digest and publishes by rename. A corrupt or unverified install never loads; the previous verified install stays until a replacement verifies.

## Individually published speech files

`source.format: "files"` supports upstream exports without an archive. Each file carries a revision-pinned `url`, `member`, SHA-256 and size. `source.sha256` is the SHA-256 of Go's canonical `json.Marshal(artifact.Files)` inventory (sorted role keys, File struct field order), and `source.sizeBytes` is the sum of payload sizes. Catalog validation binds both; provisioning reuses the verified individual-file path used by rewriters and publishes atomically. Existing archive entries keep their original semantics.

The experimental `xasr-160-qwen3asr-17-final` bundle pairs X-ASR streaming previews with Qwen3-ASR 1.7B INT8 finalization on macOS arm64. Its files come from the community ONNX export linked by sherpa-onnx; source checkpoint parity remains unverified. Use `arion-models ensure -artifact qwen3-asr-1.7b-int8 -root <cache>` to provision it explicitly (`-offline` forbids downloads). A completed, paired development report is required before catalog evaluation/default eligibility.

## Bundles and evaluations

A bundle pairs a streaming preview artifact with an optional acoustic finalizer and fixes the pipeline settings it was measured with. `evaluations` holds content-free summaries copied from completed benchmark reports by `scripts/catalog-evaluation.mjs`: run ID, report path, hardware class, split, per-language error rates, warm stop-to-review and first-partial p95, real-time factor and observation count. The app loads the best evaluated bundle by default; unevaluated bundles are never chosen.

## Adding an artifact

1. Download the archive from its immutable release URL into `.workspace/`.
2. `go run ./cmd/arion-models inspect -archive <file>` (from `apps/desktop`) prints the archive digest and every member's size and SHA-256.
3. Record the entry, licenses and conversion provenance; `go test ./...` in `models/` rejects missing provenance.
4. Benchmark bundles that use it and record their evaluation summaries before they can become the default.
