# Natural ASR source text review

All120 historical public read-speech source/reference pairs personally inspected before any new candidate inference on this corpus. This is an assistant text-only audit, not audio rehearing, independent human ratings or authoritative correction of the published reference.

The ASR source includes omitted names, numerical digits and long trailing clauses, as well as a positive question where the reference contains negation. Public references themselves contain apparent orthographic/lexical artifacts. Evaluate faithful source editing separately from evidence-supported recovery against spoken reference. Without context, hidden names/numbers/omitted clauses cannot be demanded from a rewriter. No new reference-derived OCR is invented and no evaluator text may enter model input. Keep all120, including ambiguous or severely corrupted sources; do not select only readable cases.

`verify.py` checks complete120 coverage, exact source/reference hashes and existing speaker/split identity. It does not validate the subjective notes as truth. Original notes are evaluator-side; all independent-rating and actual audio/native requirements remain pending. Historical sources are regression evidence, not unseen spontaneous dictation.

```sh
python3 benchmarks/reproduction/rewrite-natural-text-review-v1/verify.py
```

`prepare.py` preserves the exact authoring source; it writes the original review packet and must not be run over an existing archived packet. Reproducing hashes uses the verifier, never relabeling old observations. No ASR or LLM inference occurs here.

The actual archived production CleanSpeech/ApplySpeechSpans implementation was also executed on all120 sources in an isolated Go module. All120 outputs are unchanged, personally compared and retained with source/command hashes in `cleanup-execution.json`. This adds deterministic cleanup observations, zero ASR/LLM generations, and no timing/native-chain acceptance.
