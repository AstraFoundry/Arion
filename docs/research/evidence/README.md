# Evidence provenance

- [Typeless ledger](typeless-static.json) — compact derivative of actual REA analysis plus ordinary shell artifact findings; includes original Evidence IDs, SHA-256, offsets and coverage limitations.
- [Doubao ledger](doubao-static.json) — ordinary shell static observations; locally assigned DS identifiers, not REA Evidence.
- [Phonetic ASR ledger](phonetic-asr.json) — matched REA artifact identities, bounded phonetic/N-best searches, excluded TTS/tokenizer false leads and reused Chatterfly payload instructions.
- [Doubao decompiled ledger](doubao-decompiled.json) — REA 6.4.0 + Ghidra 12.1.4 Evidence IDs for five decompiled main-executable functions (insertion gates, clipboard handoff, commit strategy, fn long-press), the reference-recovery method and unresolved questions. Decompiled code itself is not committed.
- [Chatterfly ledger](chatterfly-rewrite.json) — installer/archive identities and focused native post-processing evidence with explicit limitations.
- [Keyboard IME ledger](keyboard-ime.json) — matched REA identities and registration, byte-verified Doubao/Chatterfly Pinyin decoding/ranking, model/cloud branches and candidate/commit paths; local static IDs are distinct from REA Evidence IDs.
- [Doubao disfluency ledger](doubao-disfluency.json) — byte-verified decoder annotation cleanup and bounded organization builder/result observations; a timed-out REA query produced no pseudocode.
- [Chatterfly disfluency ledger](chatterfly-disfluency.json) — byte-verified short-input, result-selection and language-policy follow-up with reused REA artifact identity.
- [Onboarding ledger](competitor-onboarding.json) — Typeless flow/gate source ranges, Doubao native page/persistence surfaces and byte-verified Chatterfly installer transitions; includes Arion source/test gaps.
- [Focused competitor rewrite ledger](competitor-rewrite.json) — original Typeless source ranges and Doubao strings/reflection observations; CR identifiers are local observations, not REA Evidence IDs.
- [Rewrite reliability ledger](rewrite-alignment.json) — reverified Doubao/Chatterfly identities, reused evidence and content-free local text diagnostic results; no competitor runtime comparison.
- [Multilingual disfluency probe](multilingual-disfluency.json) — developer-authored language-stratified text checks, quoted-filler rule repair and an exploratory model-size comparison; no additional recognition-language support claim.
- [Edit learning ledger](edit-learning.json) — Typeless, Doubao and Chatterfly correction tracking, storage and recall literals with byte offsets; EL identifiers are local static observations, not REA Evidence IDs.
- [Competitor window ledger](competitor-windows.json) — Typeless full-screen/workspace configuration and Doubao native ASR/tip panel hierarchy, numeric arguments and bounded instruction locations; CW identifiers are local static observations, not REA Evidence IDs.

The Typeless complete REA record and REA summary view, and the Doubao REA records with their scan scripts (`.workspace/research/doubao-rea/`), are retained in ignored `.workspace/research/` on this machine. The compact ledger is not a portable replacement for the complete REA record and cannot be used as an inline Evidence input. Reproduce against the documented artifact digest and retain a fresh full record when continued graph analysis is needed.

Neither report inspected private application databases, recorded user speech, or intercepted live service requests. Sanitized ledgers contain names and bounded static facts, not commercial code or model weights.
