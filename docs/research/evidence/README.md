# Evidence provenance

- [Typeless ledger](typeless-static.json) — compact derivative of actual REA analysis plus ordinary shell artifact findings; includes original Evidence IDs, SHA-256, offsets and coverage limitations.
- [Doubao ledger](doubao-static.json) — ordinary shell static observations; locally assigned DS identifiers, not REA Evidence.
- [Doubao decompiled ledger](doubao-decompiled.json) — REA 6.4.0 + Ghidra 12.1.4 Evidence IDs for five decompiled main-executable functions (insertion gates, clipboard handoff, commit strategy, fn long-press), the reference-recovery method and unresolved questions. Decompiled code itself is not committed.

The Typeless complete REA record and REA summary view, and the Doubao REA records with their scan scripts (`.workspace/research/doubao-rea/`), are retained in ignored `.workspace/research/` on this machine. The compact ledger is not a portable replacement for the complete REA record and cannot be used as an inline Evidence input. Reproduce against the documented artifact digest and retain a fresh full record when continued graph analysis is needed.

Neither report inspected private application databases, recorded user speech, or intercepted live service requests. Sanitized ledgers contain names and bounded static facts, not commercial code or model weights.
