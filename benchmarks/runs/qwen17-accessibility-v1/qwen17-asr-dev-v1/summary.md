# Benchmark report

Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| xasr-160-xasr-final:warm | 118 | 0.0709 | 0.0371 | 0.2527 | 0.5043 | 0.5043 | 136.3622 | 0.1872 |
| xasr-160-qwen3asr-17-final:warm | 118 | 0.0630 | 0.0258 | 0.2826 | 0.5565 | 0.5565 | 1820.4654 | 0.4205 |

Per-language/condition results, paired confidence intervals, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
