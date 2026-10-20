# Selected rewrite product integration observations

The signed/notarized local Arion installation now loads `qwen3.5-2b-q4-k-m-metal` and reports ready. The exact shipped recipe is `selector-screen-v1-baseline+work-scope-v4`. The user accepted the existing baseline's known OCR-choice errors; no case-specific prompt rules were added during integration.

- 457 producer preparations, 101 actual selector payloads and 16 formatter payloads match the evaluated recipes.
- Core/application, model catalog, LLM/client/provisioning tests and vet pass. The full desktop check, including native keyboard tests, passes. The final application package passes signature, notarization, architecture/resource and minimum-OS metadata verification.
- Seven initial development-worker texts and two sets of five packaged-worker texts were executed and personally reviewed: [all 17 returned bodies with scene labels](review.md), [bound review records](review.json). These include repeated controls and are not 17 independent held-out cases.
- The final packaged-worker test passes session-erasure parity, zero-generation preparation, approximately 21ms cancellation and service after cancellation. The installed app's native child has a private mode-0700 Unix socket, no TCP/UDP listener and no saved KV files: [read-only process observation](final-product-observation.json).
- Retired research/correction weights plus the installed old ONNX model were removed: 20,578,015,959 logical file bytes (about 20.6GB / 19.2GiB). ASR artifacts and original model-comparison evidence remain. Selected development/product weight paths share one verified local artifact through a hard link; weights are outside Git and the signed bundle.

[Summary](summary.json), [final native observations](final-native-observations.jsonl), [final native test log](final-native-tests.log), [desktop checks](desktop-check.log), [final package build](final-build.log), [retired research weights](removed-weights.json), [retired installed model](retired-installed-model.json).

## Scope and failed attempts

Actual host: M4/32GB. No new ASR, OS OCR, foreign-editor insertion, blind independent review, 16GB host or older-macOS execution is claimed. Context tests supply text fixtures. `peakRssBytes` in worker responses measures the Go worker and excludes its native child; it is not total model memory.

The first slot-erase setup attempt failed HTTP501 before generation; the pinned runtime requires enabling its slot-action capability through a private save-path directory. No save/restore request is issued. The initial smoke preparation sampled one token despite `max_tokens=0`; the shipped preparation now renders/tokenizes only and returns zero output tokens. Original failed/raw observations remain archived. A final equivalent producer index and configured-token-cap guard preserve all parity fixtures and pass the final native test; templates/model are unchanged.

The full raw inputs/outputs, review, receipts and archived source copies are hash-bound by `archive.json`. [Offline verifier and reproduction notes](../../reproduction/rewrite-product-integration-v1/README.md). Verification replays recorded evidence, not inference; retired models require their pinned weights to be downloaded again.
