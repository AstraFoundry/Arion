# Recorded experiment evidence

These directories retain actual per-case outputs, source/prompt observations, warmups, failure records, timings and logs. See the [archive map](../migration/archive.json) for byte identities and original locations, the [corpora index](../corpora/README.md) for inputs and the [benchmark index](../README.md) for findings.

| Study | Evidence |
| --- | --- |
| `disfluency-alignment/` | Matched baseline/product arms, rejected span/prompt probes, freeze and build manifests |
| `rewrite-alignment/`, `multilingual-disfluency/` | Earlier reliability probes, model comparisons and regression checks |
| `live-rewrite/` | Original live text and streaming/final audio observations |
| `context-ablation/`, `scene-switching/` | Context/OCR and automatic-scene observations, including development iterations |
| `edit-learning/` | Native learning/recall measurements and separate test ASR observations |
| `model-selection/` | Earlier incomplete attempt and completed bundle comparison |
| `qwen-diagnostics/` | Text, thinking, two-pass, frozen-ASR, model, prefix and acoustic diagnostics, plus disclosed review tasks/mappings |
| `phonetic-asr/` | Failed first attempt, cancelled full matrix, completed 1.7B view, preflight and offline acoustic control |
| `user-modify-asr-v1/` | Qwen3-ASR-1.7B audio-stage five-arm comparison, preflight, warmups and acquisition failures; identities in the [study archive](../results/user-modify-asr-v1/archive.json) |
| `user-modify-asr-v2/` | Development ablations, frozen policy selection and final recognition-stage retrieval comparison; actual calls, shared first-pass observations, paper sources and identities in the [study archive](../results/user-modify-asr-v2/archive.json) |
| `user-modify-asr-v3/` | Empty-vault teaching, encrypted-file/reopen snapshots, later audio inference and pre-inference split failures; [study archive](../results/user-modify-asr-v3/archive.json). Private vault keys are excluded. |
| `context-memory-v1/` | Actual text-model generation, prior corrections, rendered product/research prompts and validator decisions on reconstructed context; [study archive](../results/context-memory-v1/archive.json). No genuine OCR/audio execution is implied. |

`phonetic-asr/phonetic-asr-v2/completed-17b-view` is an exact subset of the original cancelled run. The archive keeps all excluded 4B observations; the completed report does not score them. `model-selection/models-dev-1` retains its original incomplete `running` metadata. Do not change historical statuses to make an archive check appear complete.

Review tasks and mapping files are now disclosed historical material. Their existence does not establish completed independent ratings. Migrated observations remain historical; the separately indexed new study has its own provenance. None of these experiments establishes product release acceptance.
