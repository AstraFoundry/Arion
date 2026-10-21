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
| [user-modify-asr-v1](user-modify-asr-v1/provenance.json) | `corpus.json`: 32; prior memory, separate-voice warmup and frozen model metadata | 71 |
| [user-modify-asr-v2](user-modify-asr-v2/test/provenance.json) | Development: 32 exposed cases; test: 38 new authored inputs plus 20 historical natural-read controls; unchanged prior memory and pre-development freeze | 145 |
| [user-modify-asr-v3](user-modify-asr-v3/test/provenance.json) | Development: 32 earlier audio + 66 future inputs; test: 48 earlier audio + 116 future inputs; entity/voice/session/template separation, empty initial memory | 519 |
| [context-memory-v1](context-memory-v1/protocol.json) | Development: 12 context branches/controls; test: 16; reported/authored ASR strings and reconstructed screen text, no audio/OCR capture | 9 |

The count is retained files, not independent cases. Audio manifests can describe the same cases as a text corpus. Context and scene directories include authored renderings, builders are archived separately, and audio retains the recorded waveform rather than relying on TTS regeneration. Missing licence fields in authored cases are recorded as unspecified in sidecars; the migration does not rewrite their original bytes.

Use `pnpm bench:archive:verify` for file/audio hashes and `pnpm bench:archive:report --out <new-directory>` to regenerate the supported reports without inference. Model/runtime provisioning is separate.

[correction-chain-v1](correction-chain-v1/manifest.json) adds 24 developer-authored
synthetic speech challenges, two strict silence controls and five separate-voice
development fixtures. `inputs.json` contains no references or outcome constraints;
`labels.json` is evaluator-only. Its own manifest pins the original audio and labels;
it is independently verified and is not part of the historical migration manifest.

[imk-context-v1](imk-context-v1/README.md) contains 50 original context/target homophone pairs for context ranking.

[imk-quality-v1](imk-quality-v1/README.md) contains 100 original synthetic full-Pinyin/abbreviation/v/ü/coined-phrase inputs with evaluator-only targets and explicit provenance. Revision 2 corrects one authored spelling typo; both frozen revisions and complete runs are preserved separately.

[rewrite-history-v1](rewrite-history-v1/README.md) retains401 exact historical source/context/scene tuples and original row/file lineage; its enumeration snapshot is evaluator-only.

[rewrite-work-scope-v3](rewrite-work-scope-v3/README.md) adds16 locally authored source-only work controls for status, owner/deadline scope, conditional gates and genuine versus quoted/meaningful repetition. No audio, cleanup, OCR or spelling selection is executed for these controls.

[rewrite-tone-controls-v1](rewrite-tone-controls-v1/README.md) contains24 locally authored synthetic text/context pairs testing tone-related typos, semantic-role conflicts, exact/quoted spelling, English letter evidence, ambiguity, malicious facts and absent/unrelated context. No new audio or Vision capture.

- [Semantic roles](rewrite-semantic-roles-v1/README.md): eight fixed source groups with four context conditions; authored evaluator frozen before inference.

- [Selector-screen evaluator](rewrite-selector-screen-v1/README.md):101 preregistered source/reference-bound choice targets; same-letter canonicalization remains optional, evaluator never enters model data.

- [asr-window-v1](asr-window-v1/README.md) — twelve authored Tingting utterances with original AIFF and hash-bound WAV for paired native ASR/window-context/final-rewrite integration.
