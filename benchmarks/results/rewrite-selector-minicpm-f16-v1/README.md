# MiniCPM5-1B F16 comparison

Both configurations are rejected: each conforms on82/101 choices; harmful changes are15 and17. Required repairs improve to43/47 and45/47, but unsupported OCR replacements increase. Against the prior Q4 artifact,25 pairs improve,18 regress and4 are allowed normalization differences.

All202 choices,12 native responses,22 unique failed tuples and47 changed pairs were personally reviewed. This is development review, not independent release acceptance. No new final bodies, ASR, Vision or production integration; actual Apple Silicon16GB acceptance remains open.

See [review](review.md), [machine summary](summary.json), [protocol and reproduction](../../reproduction/rewrite-selector-minicpm-f16-v1/README.md) and [raw evidence](../../runs/rewrite-selector-minicpm-f16-v1/README.md). Replay and archive verification require no model weights.
