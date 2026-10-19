# Semantic reassessment with flexible intent rewriting

Status: all 156 original delivered outputs reviewed by the Codex agent. This is
one non-blinded, agent-authored semantic review after seeing previous results,
not independent human ratings, a new held-out evaluation or new model inference.
Original outputs, fixture labels, mechanical scores and archive bytes are unchanged.

## Revised interpretation

The old exact-string gate was too narrow to represent intent quality. Equivalent
wording, capitalization, numeral spelling, useful restructuring and omission of
optional thanks are allowed. Numbered lists are not mandatory when clear prose
retains the same items. Conversely, a keyword pass does not establish fidelity:
all six outputs preserve the uncertainty phrase in the cache example, but change
its subject to cash and therefore change the meaning.

| Pipeline | Intent preserved | Degraded | Uncertain | Ready to use |
| --- | ---: | ---: | ---: | ---: |
| Streaming ASR | 21/26 | 4 | 1 | 12/26 |
| Streaming ASR -> correction | 20/26 | 5 | 1 | 12/26 |
| Streaming ASR -> rewrite | 20/26 | 6 | 0 | 16/26 |
| Streaming ASR -> correction -> rewrite | 20/26 | 5 | 1 | 17/26 |
| Offline acoustic control | 23/26 | 1 | 2 | 13/26 |
| Offline acoustic control -> rewrite | 22/26 | 2 | 2 | 19/26 |

All denominators include two silence cases. Intent preservation and organization
are separate. A raw spoken repair can preserve the final value but still need
cleanup; fluent output can be wrong. Ready requires preserved intent plus adequate
readability. Organization is an ordinal 0/1/2 annotation, not a weighted semantic
score. No inter-rater reliability or statistical generalization is established.

The dedicated correction stage introduces one clear additional semantic loss:
it drops Robert and retains the superseded recipient Helen. Of the three formerly
correct sources whose word/character edit count increased, the meeting's ding
spelling and removal of optional thanks are not semantic damage under this rubric.

For the full chain versus direct rewriting, one final-day error is fixed and one
recipient is lost. Readiness gains are the final-day and budget repairs; the loss
is a residual English huh filler. The net 17 versus 16 is a small mixed effect,
not evidence of a meaningful advantage. The stronger earlier claim that all three
edit-distance regressions were harmful is withdrawn.

## Examples of changed judgments

- The shortened Nora email is acceptable: it retains the recipient, receipt of
  the draft and tomorrow's review. Missing optional thanks is not failure.
- The plain first/second/third list remains acceptable without `1.`/`2.`/`3.`.
  Numbered lines receive better organization ratings, not exclusive correctness.
- `A P I` and `Mac O S` preserve the same letter sequences and platform contrast.
  Joining them improves presentation; it is not recovery of a lost fact.
- `cache` -> `cash` is a material change even though the prior required uncertainty
  strings still occur.
- Missing self-repair punctuation in the Wang/Chen example is marked uncertain,
  rather than confidently wrong, until the direct rewrite explicitly reinforces
  the wrong recipient. The explicit offline repair is preserved.

Boundary judgments are exposed in [ratings.json](ratings.json): `pool request`
inside review context is treated as a recognizable minor spelling artifact,
whereas `good request` and the phonetic database spelling remain uncertain. A
stricter technical-identifier rubric could change those rows. Every rating has
its own output digest and a case-specific rationale; nothing derives semantic
labels from WER, the old keyword gate or model identity.

## FastCorrect acquisition retry

The official checkpoint and its SentencePiece model now download successfully
through the existing macOS system proxy, explicitly supplied to curl. The earlier
failure used a direct connection. The repository dictionary was also downloaded
at a pinned Git revision and verified against its Git blob SHA-1.

- `fastcorrect_pretrain.pt`: 914,618,147 bytes (about 872 MiB).
- SentencePiece model: 543,619 bytes, parsed successfully, 20,000 pieces.
- Character dictionary: 32,652 entries; plus Fairseq's four specials matches the
  checkpoint's 32,656-row embeddings.
- All 342 ZIP entries pass CRC checks. Restricted `torch.load(weights_only=True)`
  with only `argparse.Namespace` explicitly allowlisted reads the checkpoint.
- Architecture metadata reports six encoder and six decoder layers, width 512.
  State-tensor counts are not an instantiated parameter count because shared
  embeddings and buffers require model-level accounting.

Weights remain in `.workspace/correction-chain-reassessment/models/fastcorrect/`.
The official artifact is a **pretraining** checkpoint; full FastCorrect inference
and AISHELL fine-tuning have not been executed. The custom legacy Fairseq model
and dictionary/tokenizer integration still need validation. Download and structural
inspection are not a successful benchmark or a reproduced paper score. No publisher
SHA-256 was supplied for the Drive files; acquired-byte hashes pin this download.
[Acquisition and inspection metadata](../../runs/correction-chain-semantic-v1/downloads/fastcorrect-manifest.json)

## Reproduce and inspect

- [Rubric](../../reproduction/correction-chain-semantic-v1/rubric.json)
- [Per-output annotations](ratings.json)
- [Summary, paired changes and reclassified cases](summary.json)
- [Independent derivative archive](../../runs/correction-chain-semantic-v1/manifest.json)

```sh
pnpm chain:semantic-report .workspace/semantic-replay-new.json
pnpm chain:semantic-test
pnpm chain:semantic-verify
node benchmarks/reproduction/correction-chain-v1/archive.mjs verify
```

This replays the annotations and checks exact output/rubric identities without
loading a model. It does not automatically produce independent semantic judgments.
