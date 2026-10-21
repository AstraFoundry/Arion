# Benchmark report

Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| xasr-160-xasr-final:warm | 118 | 0.0709 | 0.0371 | 0.2527 | 0.5043 | 0.5043 | 145.5955 | 0.1815 |
| xasr-160-qwen3asr-17-final:warm | 118 | 0.2472 | 0.0645 | 0.3478 | 0.4696 | 0.4696 | 1851.6024 | 0.4027 |

Per-language/condition results, paired confidence intervals, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
