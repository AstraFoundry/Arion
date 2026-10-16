# Recognition-stage execution

- Qwen3-ASR-1.7B community MLX 8-bit is a new audio model; the existing text rewriter is not used.
- Frozen initial approved memory is authored independently of test audio or model outputs.
- All primary arms receive identical audio and checkpoint. No test reference reaches inference or selection.
- Empty context, selected destination words, selected correction pairs, tagged selected pairs, and unfiltered pairs are compared.
- Declared Doubao-style selection: 30-day expiry, reverse-pair suppression, deduplication; today then recall then use; total 8 entries, 512 Unicode characters. Bucket quotas, timestamp ties and serialization are experimental values because competitor values are unknown.
- Memory is frozen during paired inference; updating counts is separately checked by a deterministic sequential policy simulation, not a claim of live product learning.
- Stop after one frozen run; report raw recognition and error increases. Do not tune against test outputs.
- Quantized export/source checkpoint parity, natural speech, independent ratings, microphone streaming and native integration remain unverified.
