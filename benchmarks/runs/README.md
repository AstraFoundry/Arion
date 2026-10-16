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

`phonetic-asr/phonetic-asr-v2/completed-17b-view` is an exact subset of the original cancelled run. The archive keeps all excluded 4B observations; the completed report does not score them. `model-selection/models-dev-1` retains its original incomplete `running` metadata. Do not change historical statuses to make an archive check appear complete.

Review tasks and mapping files are now disclosed historical material. Their existence does not establish completed independent ratings. None of these files is a new inference run or proof of product release acceptance.
