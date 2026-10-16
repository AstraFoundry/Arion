# Benchmark report

Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| xasr-160:warm | 118 | 0.0961 | 0.0613 | 0.3587 | 0.3652 | 0.3652 | 1019.3725 | 0.9266 |
| xasr-480:warm | 118 | 0.1260 | 0.0677 | 0.3370 | 0.1913 | 0.1913 | 44.2252 | 0.0926 |
| xasr-160-xasr-final:warm | 118 | 0.0709 | 0.0371 | 0.2527 | 0.5043 | 0.5043 | 189.8405 | 0.2248 |
| xasr-480-xasr-final:warm | 118 | 0.0709 | 0.0371 | 0.2527 | 0.5043 | 0.5043 | 156.7450 | 0.1085 |
| xasr-160-sensevoice-final:warm | 118 | 0.0724 | 0.1306 | 0.3804 | 0.3478 | 0.3478 | 248.3387 | 0.2561 |
| zipformer-2023:warm | 118 | 0.3748 | 0.1065 | 1.2826 | 0.1478 | 0.1478 | 50.8605 | 0.1016 |

Per-language/condition results, paired confidence intervals, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
