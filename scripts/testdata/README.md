# Public regression fixtures

[public-fixtures.json](public-fixtures.json) records the original public source
revision, archive paths, sizes, SHA-256 identities, attribution and current license
locations for the product-owned regression inputs. The fixture, original
provenance and original README bytes are preserved unchanged. Their historical
paths identify the original archive; product tests use the public paths in this
manifest and do not open the private evaluation repository.

The inputs belong beside the tests they protect:

- Prompt payload, original-template and spelling-producer parity live under the
  corresponding desktop package's `testdata`.
- The native GGUF smoke inputs remain explicit opt-in text probes. They do not
  establish speech accuracy or model quality.
- Shared IME Unicode, legacy-storage and English-admission fixtures live under
  `apps/desktop/ime/shared/testdata`. Exposed admission sets remain regression
  material, including the historically named holdout.
- The two Windows packaged-speech smoke WAVs live under
  `apps/desktop/e2e/testdata`. The LibriSpeech attribution and license are recorded
  in the manifest; generated silence retains its original corpus provenance.
- [The pinned Rime source](ime-spelling-reference-v1/README.md) exercises the
  exact production correction patch and retains its original BSD license.

Verify all retained bytes without models or private files:

```sh
node scripts/verify-public-fixtures.mjs
```

Historical provenance may contain paths relative to its former archive location.
The manifest's `licensePath` fields resolve the current public notices without
rewriting those original records.
