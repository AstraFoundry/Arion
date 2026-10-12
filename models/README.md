# Model catalog

Do not commit weights. This foundation has no approved catalog entries. [006 Models](../specs/006-models.md) defines the required metadata and release gates.

An entry must record artifact ID/purpose; exact upstream revision and conversion provenance; SHA-256/byte size/source URL; runtime and checkpoint license references; runtime version/platform/architecture; quantization; tokenizer and typed-head scoring semantics; supported languages; audio rate/channels for ASR; benchmark evidence. Omit inapplicable fields explicitly, never invent hashes or use `latest` as provenance.

The initial evaluation tasks compare streaming speech runtimes and Laya Multilingual Jev-style Choice. A successful download is not distribution approval. User data and benchmark audio belong outside version control unless deliberately consented and licensed.
