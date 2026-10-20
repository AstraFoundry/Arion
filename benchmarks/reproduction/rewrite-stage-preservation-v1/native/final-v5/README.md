# Final v5 native evidence

Actual Vision, production ASR/coordinator and source-built v5 worker run on frozen synthetic fixtures. See [summary.json](summary.json) for exact outcomes and [manifest.json](manifest.json) for original paths and file hashes. Runtime executables and model weights are excluded; their identities remain in build/invocation metadata. Source Markdown is preserved byte-for-byte as `.md.txt` evidence with explicit mappings in build provenance.

All three Chatterfly reproduction finals are correct. Original case 01 and its screen-off duplicate have validated correct previews; case 02 has a cancelled live request and a corrected final. The ten-second-silence cache pair has identical complete effective requests except Live and identical raw output: cache-on performs zero final worker calls, cache-off performs one. The five-second pilot is retained with its scheduling confound.

The technical regression restores getUserProfile but adds “不要补充注释”, reversing the intended positive action after ambiguous ASR “之补充注释”. It is a semantic failure, not a passing pipeline case. Competing-contact disambiguation abstains and leaves the ASR name unresolved. Both numeric contrasts are preserved.

These are exposed development diagnostics. Approved memory was constructed after earlier observations; duplicated speech and authored silence are not independent test samples. No held-out quality, typical latency, native insertion or Windows claim follows.

[Fixture provenance](external-fixture-provenance.json) resolves reused PNG/OCR evidence to preserved sibling archives with explicit hashes and verifies every actual worker screen-context string against its corpus input and observed accurate OCR. Original corpus image fields remain unchanged.
