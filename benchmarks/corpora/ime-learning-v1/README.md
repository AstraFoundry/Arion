# Shared IME learning fixtures

`legacy-nsdata-pairs.bplist` is an original synthetic array of three binary data pairs, matching the original macOS `NSPropertyListSerialization` format: an embedded-NUL key/value, repeated data (exercising plist object deduplication), and a UTF-8 Pinyin/Chinese term. It contains no personal text. It was independently encoded using Python standard-library `plistlib` in binary mode. The shared encoder's output is checked by the independent reader as a mapping, because row order has no database meaning.

`GraphemeBreakTest-17.0.0.txt` is the original Unicode Consortium extended-grapheme conformance corpus, unchanged. Source URL, SHA-256 and license are recorded in `apps/desktop/ime/shared/ranking_unicode_sources.json` and `ranking_unicode_LICENSE.txt`. Unicode property inputs regenerate the checked-in table with `ranking_unicode_generate.py INPUT_DIRECTORY`; the generator does not download data.

`Scripts-17.0.0.txt` retains the original official Unicode 17 Script property
bytes. The shared ranker test parses it independently and compares Han detection
for every Unicode scalar, including unlisted Unknown values, through both C++
and the length-aware C bridge. Additional cases cover Latin/non-Han scripts,
mixed and supplementary Han, embedded NUL and malformed UTF-8 before/after Han.
Supply all files listed in `ranking_unicode_sources.json`, including
`Scripts.txt`, to the offline generator; it refuses any source hash mismatch.
The Windows module fixture separately checks that non-Han/invalid commits leave
the actual encrypted context file byte-for-byte unchanged, mixed/supplementary
Han confirmations promote candidates, and a failed runtime rejects even non-Han
no-ops.

Run the common checks on native macOS or Windows after explicitly staging the pinned keyboard SDK:

```sh
pnpm keyboard:stage
pnpm keyboard:shared:check
```

The shared entry runs the C session/input/menu boundary tests, then `scripts/test-shared-keyboard-learning.mjs`. That runner compiles the shared codec and ranker with strict C++17 warnings, verifies the fixture and Unicode source-ledger hashes, runs all 766 Unicode conformance cases, and uses an independent Python 3 standard-library `plistlib` reader to check the emitted legacy mapping. Set `CXX` or `PYTHON` to select the compiler or reader explicitly.

On Windows, the learning runner also exercises actual current-user DPAPI persistence and the production encrypted Rime database/cursor/transaction/context ABI. Its owned child-process matrix checks overlapping Rime sessions, confirmation counts/global clocks/dynamic weights, different words, pending abort, terminated pending/writing processes, readonly refresh, context ranking and live-lease reset refusal. The native mode-settings fixture checks atomic field toggles in two processes, failure preservation and actual ACL/hardlink/junction rejection. It requires the same MinGW-w64 GCC ABI as the staged SDK and an ordinary medium-integrity desktop token. Each storage/ABI test receives a separate newly owned profile under `.workspace/tests/shared-keyboard-learning/run-*/profiles`; it never opens a user's learning vault. All spawned fixture processes belong to an owned job with kill-on-close cleanup. A failed stage, fixture, compile or native check fails the command.

`report.json` in that run directory records source/fixture/SDK/runtime and executable hashes, compiler identity, exact commands, status and raw test outputs, including failed attempts. These are storage/ranking correctness checks, not conversion-quality, latency or installed-editor evidence. macOS runs the portable codec and ranker here; `pnpm keyboard:check` retains its separate native learning and AppKit tests. This portable runner does not establish macOS Keychain behavior.

`pnpm keyboard:windows:check` builds the Windows engine/module and performs package validation, then runs the shared checks and the actual Rime conversion fixture. It does not install a TSF profile or establish editor dispatch acceptance.
