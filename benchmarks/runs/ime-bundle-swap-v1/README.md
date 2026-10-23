# Directory exchange validation observations

- `0a65813e-9322-4081-8a90-72d3f607cc88/result.json`: initial x86_64-process run on macOS 26.6.2, 2000 successful exchanges and 104292 destination-stat observations, no observed missing destination. This does not claim Intel hardware coverage.
- `225f0f5e-cb60-4346-9f32-6b44b024930c/result.json`: explicit native arm64 run on the same OS, 2000 successful exchanges and 160955 destination-stat observations, no observed missing destination.
- Both runs verify reverse exchange restores hashes and a missing-target error preserves both complete directories. Both fixture locations are on the same device as the installed input-method parent.
- `volume.json`: native Foundation/statfs observation identifies APFS and swap capability.

These are repeated filesystem validations, not independent hardware samples or performance comparisons. No signed production bundle, unsupported volume, power-loss or actual system switching test ran. Sources and original-byte identities: [reproduction](../../reproduction/ime-bundle-swap-v1/README.md).
