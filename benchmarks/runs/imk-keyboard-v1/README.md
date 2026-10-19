# Offline IMK keyboard functional evidence

The completed latest run passes **20 / 20** real librime 1.17.0 and AppKit text-store cases. The separate system registration attempts remain **failed**: neither establishes actual IMK typing dispatch. This archive contains synthetic text only, original successful and failed outputs, immutable artifact provenance, matching executed sources and restoration records.

Run the read-only, model-free check from any checkout:

```sh
node benchmarks/runs/imk-keyboard-v1/verify.mjs
```

`archive.json` maps every archived file to its original path, bytes and SHA-256. The final attempt's recorded source identities were checked against current bytes before copying; its harness, production engine/typing/encryption sources, compiled schema and runtime manifest are preserved exactly. The native library is represented by its observed SHA-256 and public pinned acquisition manifest, not copied. Supporting headers/staging scripts are labelled as archive-time snapshots where no separate execution-time digest was recorded.

## Functional boundary

`engine/typing-attempt-11.json` and `.log` retain the completed 20 cases: full Pinyin; apostrophe, abbreviation and v/ü; raw Enter and Chinese punctuation; candidate paging and number selection; partial selection/suffix conversion and rollback; cursor/Home/End/Delete; cancellation, deactivation and stale callbacks; emoji/CJK neighbors; restart learning and cancellation; data integrity failure; and actual storage failure preserving an already produced text commit.

The client is a protocol adapter around real `NSTextView`; conversion uses actual independently licensed librime and pinned data. Encryption uses the production backend with a synthetic isolated key. The run does not access Keychain or register/select an input source. Tests show 事实 promoted from second to first for `shishi` after actual engine restart. Cancelling raw and partial compositions leaves the complete decrypted learning map and ticks unchanged. Another case obstructs the isolated vault path with a regular file, provoking a real failed Rime transaction flush while verifying the actual engine commit reaches the host exactly once.

Earlier attempts retain their original failed or passing status. Attempts 01, 07 and 09 did not compile. Attempts 02, 03 and 06 failed assertions; their outputs remain unedited. Attempt 03's historical text trace used mutable AppKit backing strings and is explicitly limited; attempt 04 fixes capture immutability. Exact historical harness revisions before attempt 11 were not saved. Their raw outputs and available identities are retained without claiming a complete byte-identical rerun for those revisions. See `engine/typing-test-report.md` for source timing, fixes and limitations.

## Reproduce the final functional harness

Use macOS with Xcode tools and the pinned public runtime/data. Runtime, SDK and generated storage must remain in `.workspace/`, outside this archive. Copy the frozen source tree into a new workspace, stage pinned dependencies there and run the harness:

```sh
replay_dir="$(mktemp -d "$PWD/.workspace/imk-keyboard-replay.XXXXXX")"
cp -R benchmarks/runs/imk-keyboard-v1/sources/. "$replay_dir/"
node "$replay_dir/scripts/stage-keyboard-runtime.mjs"
node "$replay_dir/scripts/test-keyboard-input.mjs" --report "$replay_dir/result.json"
```

The source snapshots retain original relative paths. Staging fetches only the pinned public archives if absent; the test runner itself never downloads or deploys schemas. This command is a new functional run with a new output path, not a replayed claim of OS acceptance or a latency benchmark. Hardware power mode was not recorded; no latency result is reported.

## Failed system registration and restoration

`system/run-03/observations.json` records the original macOS 26.6.2 arm64 attempt: API installation success did not make the parent source enabled; child selection returned `-50`, current source stayed ABC and no actual typing dispatch was established. `state.json` confirms restoration of original bundle byte inventory, ABC selection and absent test learning key, with no restoration errors. Disabled cached own-source entries remained.

Run 04 retains its original `state.json`: phase restored, ABC selected, no own-source entries in the after snapshot and no restoration errors. The failed enable/installation response was reported during the investigation but not saved as a separate original observation file; that gap remains explicit. Its source driver is retained, and this run is never counted as successful acceptance.

System records are allowlisted to the two Arion IDs, current ABC source, owned process metadata, own-bundle inventories and own-key presence status. No key bytes, personal input-source preferences, user profiles, staged application binaries or unrelated app data are archived. Original system driver source has absolute scratch paths and installation side effects; it is historical evidence, not an automatically executable acceptance command. Unfinished signed/notarized execution remains outside the archive.
