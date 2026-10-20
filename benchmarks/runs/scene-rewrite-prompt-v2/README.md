# Focused scene prompt v2 raw evidence

`dev/` retains 32 generations, complete assistant review and the selection frozen
before evaluation. `evaluation/` retains all 112 paired observations and their
complete review. Load replies and stderr streams are retained for every worker.
`report/` is a deterministic model-free replay, including all 144 reviewed pairs.
No independent human review is claimed. All source/context text is authored
synthetic fixture data. No failed attempt was discarded or overwritten.

See [results](../../results/scene-rewrite-prompt-v2/README.md) and
[reproduction](../../reproduction/scene-rewrite-prompt-v2/README.md).
`archive.json` pins all v2 evidence, scripts and results; its hashes provide
integrity checks, not an external signature or independent timestamp.
