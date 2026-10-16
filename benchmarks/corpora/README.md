# Benchmark corpora

Recorded inputs are retained byte-for-byte with references, constraints, speaker/session identities and split annotations. The [archive map](../migration/archive.json) pins every original file and its destination. Historical test cases disclosed here are regression material for future development; exposure does not invalidate the recorded comparison, but they cannot serve as an unseen future test set.

| Corpus | JSON case definitions | Retained files |
| --- | --- | ---: |
| [ablation-audio-v1](ablation-audio-v1/provenance.json) | `corpus.json`: 16 | 29 |
| [asr-v1](asr-v1/provenance.json) | `corpus.json`: 274 | 275 |
| [context-v1](context-v1/provenance.json) | `context-heldout-corpus.json`: 30; `dev/context-dev-corpus.json`: 16 | 97 |
| [disfluency-audio-v1](disfluency-audio-v1/provenance.json) | `corpus.json`: 8; `audio-fixtures/manifest.json`: 8 | 10 |
| [disfluency-fresh-v1](disfluency-fresh-v1/provenance.json) | `corpus.json`: 16 | 1 |
| [disfluency-regression-v1](disfluency-regression-v1/provenance.json) | `corpus.json`: 28 | 1 |
| [live-audio-dev-v1](live-audio-dev-v1/provenance.json) | `manifest.json`: 22 | 23 |
| [live-audio-heldout-v1](live-audio-heldout-v1/provenance.json) | `manifest.json`: 27 | 28 |
| [live-heldout-v1](live-heldout-v1/provenance.json) | `corpus.json`: 30 | 1 |
| [multilingual-disfluency-v1](multilingual-disfluency-v1/provenance.json) | `corpus.json`: 21 | 1 |
| [oral-rewrite-v1](oral-rewrite-v1/provenance.json) | `corpus.json`: 16 | 1 |
| [rewrite-dev-original-v1](rewrite-dev-original-v1/provenance.json) | `corpus.json`: 24 | 1 |
| [rewrite-dev-v1](rewrite-dev-v1/provenance.json) | `corpus.json`: 24 | 1 |
| [rewrite-frozen-asr-v1](rewrite-frozen-asr-v1/provenance.json) | `corpus.json`: 56 | 1 |
| [scene-v1](scene-v1/provenance.json) | `scene-dev-corpus.json`: 33; `scene-heldout-b-corpus.json`: 30 | 6 |

The count is retained files, not independent cases. Audio manifests can describe the same cases as a text corpus. Context and scene directories include authored renderings, builders are archived separately, and audio retains the recorded waveform rather than relying on TTS regeneration. Missing licence fields in authored cases are recorded as unspecified in sidecars; the migration does not rewrite their original bytes.

Use `pnpm bench:archive:verify` for file/audio hashes and `pnpm bench:archive:report --out <new-directory>` to regenerate the supported reports without inference. Model/runtime provisioning is separate.
