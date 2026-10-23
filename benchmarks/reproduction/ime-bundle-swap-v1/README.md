# Owned macOS bundle-directory exchange validation

These original sources exercise `renameatx_np(RENAME_SWAP | RENAME_NOFOLLOW_ANY)` using unsigned, non-executable directory fixtures. No installed bundle, input-source API, application cache or clipboard is modified. The fixture's Info.plist is a marker, not an application declaration. This is filesystem validation, not IME or crash-prevention acceptance.

Run from the repository root on macOS, with an existing `~/Library/Input Methods` directory for read-only same-volume comparison:

```sh
/usr/bin/arch -arm64 /usr/bin/python3 benchmarks/reproduction/ime-bundle-swap-v1/swap_probe.py
```

New fixtures and results are written to a unique ignored `.workspace/notes/wechat-ime-crash/lifecycle-repair/` directory. The source performs a successful swap, reverse swap, missing-target error and 2000 exchanges with a concurrent destination-stat reader. Do not use it to manipulate installed applications.

The Objective-C volume probe can be compiled with Apple's clang and Foundation; it only queries the passed directory's filesystem and swap capability. Built binaries stay in the ignored workspace.

Original raw observations: [runs](../../runs/ime-bundle-swap-v1/README.md). [Provenance](provenance.json) records exact source/result hashes and original paths. Finite stat sampling is not a proof of atomicity, power-loss durability, cross-volume behavior or source-cache coherence; the installed SDK manual supplies the operation's contract.
