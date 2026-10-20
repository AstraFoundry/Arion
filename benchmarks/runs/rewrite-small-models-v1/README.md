# Latest small-model screen evidence

[Full48-output review](report/review.md) · [Summary](report/summary.json) · [Frozen protocol](../../reproduction/rewrite-small-models-v1/protocol.json). Every raw response was personally inspected. No content validator filters these outputs. Neither model is adopted.

The24 fixed cases per model comprise8 exposed and16 previously ungenerated probes, separately reported. Text-only bundles use approximately0.75B/1.88B tensor elements and no vision encoder. Runtime, quantized export, hardware, requests and review hashes are retained. Timings use no prompt cache and cannot be compared as model-only effects against historical cached ONNX runs.

The reproducible report and evidence tests do not need model weights or original scratch folders. See the reproduction README for pinned acquisition identities and commands.
