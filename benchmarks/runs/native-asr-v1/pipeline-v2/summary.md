# Benchmark report

Single-backend validation; descriptive results without a paired treatment or significance claim.

| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| qwen17-mlx8-native:warm | 118 | 0.0535 | 0.0161 | 0.2201 | 0.6000 | 0.6000 | 1387.1722 | 0.8109 |

Per-language/condition results, comparisons where present, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.
