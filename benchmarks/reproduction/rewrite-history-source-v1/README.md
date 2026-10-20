# Historical source verification and replay

```sh
python3 benchmarks/reproduction/rewrite-history-source-v1/verify.py
python3 benchmarks/reproduction/rewrite-history-source-v1/verify-enumeration.py
python3 benchmarks/reproduction/rewrite-history-source-v1/build.py --out .workspace/rewrite-history-source-replay-new
```

All401 raw input tuples and completeness against the original timeline snapshot are verified. Replay creates a new directory and reproduces inputs.json and lineage.json bytes from archived source material; its manifest truthfully references the snapshot rather than the mutable original UI path. The exact original preparation script is retained as original-builder.py.txt. No inference, weights or private environment is required. Full original JSONL files/row hashes remain authoritative; the timeline snapshot is enumeration evidence and never model input.
