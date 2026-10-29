# Shared PR keyboard comparison inputs

The [corpus](corpus.json) freezes 113 case-sensitive input tasks for the [native comparison](../../reproduction/ime-pr-comparison-v1/README.md). [Provenance](provenance.json) identifies the original PR 40 regression splits and their exact bytes. Added examples cover technical canonical spellings, incomplete words, explicit Shift/mixed-case input and both orders of Chinese/English partial selection.

Targets belong only to the evaluator. The native driver receives raw letters, shared cancelled warmups, page size and pacing; later selections use observed indices. Canonical-case misses and case-insensitive spelling coverage are reported separately. No user text, account data, weights or recordings are included. These are authored development/regression fixtures, not independent holdout or general-user accuracy evidence. No labels or thresholds are changed after comparative execution.
