# Prompt and text-validator ablation raw evidence

`dev/` retains 32 proposals, every proposal/cleanup review and the pre-evaluation
selection. `evaluation/` retains 96 paired proposals and their complete review.
Both retain model/runtime/build/input identities, worker loads, stderr, errors
and exact cleanup values. `report/` includes the complete 384 strict/envelope/none
text deliveries, all assigned/effective scenes and prompt versions, and a view
of all 96 ungated evaluation deliveries. These are 128 actual inference calls;
policy replay does not create extra model observations.

Inputs are authored synthetic ASR-like/OCR-like text. Judgments are assistant
inspection, not independent human ratings. No native target received these
outputs. [Results](../../results/rewrite-no-validator-v1/README.md) and
[reproduction](../../reproduction/rewrite-no-validator-v1/README.md) state the
limitations and failed adoption decision. `archive.json` is a checksum manifest,
not an external signature or independent timestamp.
