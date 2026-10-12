# Streaming speech datasets

Hugging Face has useful evaluation corpora, but dataset `streaming=True` means incremental **download/iteration**, not a proven realtime recognition benchmark. [Pinned catalog](../../benchmarks/datasets.json) records exact repository commits inspected through the HF metadata API. No audio was downloaded and no access terms were accepted during this research.

## Dataset selection

| Priority | Dataset / authoritative sources | Coverage and import choice | Access/limitations |
| --- | --- | --- | --- |
| P0 | [OpenSLR LibriSpeech on HF](https://huggingface.co/datasets/openslr/librispeech_asr) | English 16 kHz audiobook speech; `all` configuration validation.clean/other and test.clean/other. Map speaker_id, chapter_id and id into namespaced fixture metadata | Card CC-BY-4.0; short segmented read speech. Useful WER/control baseline, insufficient by itself for spontaneous long-form/endpoints |
| P0 | [AISHELL-1 on HF](https://huggingface.co/datasets/AISHELL/AISHELL-1), [OpenSLR SLR33](https://www.openslr.org/33/) | Mandarin indoor read speech; authoritative dev/test speakers, CER, accents and fixed-chunk replay | HF/OpenSLR label Apache-2.0, card also mentions academic use. Preserve original license/card. HF speaker archives do not themselves define the authoritative dev/test split |
| P1 | [Earnings-22 HF](https://huggingface.co/datasets/distil-whisper/earnings22), [creator repository](https://github.com/revdotcom/speech-datasets/tree/main/earnings22) | `full` long accented English recordings for continuous state and resource drift; `chunked` only for utterance comparisons. Creator files provide speaker/timestamp/entity structure | Original creator README CC-BY-SA-4.0; HF license metadata absent. Some annotations are machine-aligned or missing; verify before latency gold. Official fstalign WER differs from our normalization |
| Conditional | [SEAME dev_man mirror](https://huggingface.co/datasets/AudioLLMs/seame_dev_man), [dev_sge mirror](https://huggingface.co/datasets/AudioLLMs/seame_dev_sge) | Mandarin-English conversational code switching; context audio and answer transcript; MER after explicit handling of special tokens | Third-party mirrors with no declared license in inspected metadata. Confirm original corpus permissions and restore speaker/session mapping before acquisition. Instruction text must not become ASR context |
| Conditional | [WenetSpeech](https://huggingface.co/datasets/wenet-e2e/wenetspeech) | Mandarin web/meeting domains; DEV_fixed, TEST_NET, TEST_MEETING; useful spontaneous/domain-shift complements | Gated. Card specifies non-commercial access alongside CC-BY wording and source-owner rights. Existing remote-code loader requires review; no unattended acquisition |
| Conditional | [GigaSpeech](https://huggingface.co/datasets/speechcolab/gigaspeech) | English audiobook/podcast/web domains; dev/test complements for spontaneous speech | Gated audio agreement restricts non-commercial research/education even though metadata says Apache-2.0. Audio terms differ from code/metadata license |

Prefer original-publisher mirrors and exact pinned revisions. Public visibility or an ungated HF endpoint does not establish audio redistribution rights. Research catalog status is not a production-use approval.

## Reproducible import

1. Choose the smallest relevant dev/test subset and archive its original card/access record. Provision audio explicitly into ignored private storage. Inspect pinned loaders; do not run unreviewed remote dataset code. Never import test labels as contextual memory.
2. Export source IDs, original speaker/session/chapter/recording identities and split mapping. Prefix identities by corpus. Keep train/dev/test separated by speaker/session and waveform hash. For unknown/multi-speaker identities, preserve conservative recording/connected-speaker groups; obtain a reviewed grouping strategy before claiming speaker-independent CIs.
3. Decode/resample deterministically to mono 16 kHz, record decoder/resampler versions/settings and original/output hashes, and derive actual decoded duration. Save manifest audio SHA-256 after conversion. Do not treat the HF cache path as a reproducible artifact identity.
4. Apply a documented source-specific annotation policy, including SEAME markers, `<inaudible>`, punctuation, abbreviations and official-vs-Arion normalization. Retain verbatim source labels privately. Correct source timestamps only with review and hash the annotation artifact.
5. Final CER/WER/MER can use ordinary utterance labels. Onset/end/stable-unit and endpoint claims require reviewed speech intervals, intended turns and optional unit timing. Existing corpus alignment is a starting point, not verified latency truth. Never use the tested VAD/model's own predictions as gold.
6. Validate with `make bench-validate CORPUS=...`. Use separate train warmups for dev evaluation, dev warmups for held-out test. Freeze the selection/export manifest, model hashes and config before each comparison.

Import/conversion is a follow-up DATA-01 implementation, not a hidden dependency in pnpm install or CI. The repo provides the target manifest validator, dataset catalog and `make datasets-check` pinned metadata verification. It does not install Python/HF loaders or acquire gated data automatically. The whole project remains pnpm-managed for JavaScript workflows; a future data converter may use an explicitly pinned runtime/tool via Make.

## Streaming replay scenarios

[Executable sweep](../../benchmarks/configs/desktop-streaming.json) covers chunk size × endpoint timeout with model/memory/ranker/correction held constant. Request effective settings are checked; chronological incremental feed and real native preview traces must be provided by the driver.

Use independently labeled groups for short clean speech, hesitations/internal pauses, final quiet syllables, CJK/English switches, names/numbers/URLs, long continuous speech and nonspeech/noise. Keep original source splits. For long-form tests, preserve actual full recording order/state. Fixed concatenation of short clips with seeded silence is a separate **synthetic stress** set: record recipe/component hashes and never describe it as naturally occurring long-form conversation. Noise/gap/jitter transformations also need fixed recipes and source license tracking.

Measure final accuracy, engine/native-preview first partial and erasure, stable-unit delay with correctness coverage, unintended endpoints/missed endpoints, stop-to-review, tail proxy, dropped samples, queue age and resource traces. Follow [exact scoring methodology](benchmark-methodology.md). Public corpora do not test application target safety, consented memory benefit, candidate UI choice or semantic correction preservation; add the [owned product scenarios](../quality/e2e.md) and consented bilingual technical dictation fixtures.
