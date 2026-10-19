# Real keyboard engine and AppKit integration

Latest completed run: `typing-attempt-13.json`, **22 / 22 passed**, exit 0. Raw stderr/stdout: `typing-attempt-13.log`.

Command:

```sh
node scripts/test-keyboard-input.mjs --arch x86_64 --report .workspace/research/imk-alignment/typing-attempt-13.json
```

Uses actual pinned librime 1.17.0 and staged `apps/desktop/resources/darwin/keyboard/data/build/arion_pinyin.schema.yaml`. Production `keyboard.m` and `typing.m` compile unchanged into the test. The client protocol adapter wraps actual `NSTextView`. Identical production encrypted userdb component is registered through `ARION_LEARNING_TESTING` with an isolated synthetic 256-bit key; this never accesses Keychain and does not prove real Keychain provisioning. Schema user dictionaries remain enabled; no diagnostic fallback schema is used. Test user storage is an isolated temporary directory removed after the run.

Covered: nihao -> 你好 through actual candidate/Space commit; Escape cancellation and UTF-16 emoji/CJK neighbors; committed neighbors; Backspace/reinsertion; page-relative number selection after PageDown; partial selection plus suffix conversion; rollback of selected prefix; Home/End and middle edits; forward Delete; Shift+Space cancellation and ASCII passthrough; stale selection/highlight revisions; deactivation and callback rejection. These are KBD-A01/A03/A04/A05/A06 engine/text-store scenarios, **not complete actual IMK input-source acceptance**.

## Failed attempts retained

- Attempt 01: compilation failed because the testing-only encrypted-registry helper is C++ declared. Runner now compiles the test translation unit as Objective-C++.
- Attempts 02 and 03: 9 / 11 passed. Two expectations incorrectly assumed characterwise Left and first-candidate selection after rollback. Actual engine traces showed Left moves to a Pinyin syllable boundary and rollback correctly restores `你hao` to raw `nihao`, while retaining highlighted candidate `你`. Tests now restore the actual removed character and explicitly choose the full candidate after rollback. They do not change engine behavior to satisfy the assertions. Attempt 03 trace text used a mutable NSTextView backing NSString; attempt 04 copies text at capture so historical traces are immutable.
- Attempt 04: 12 / 12 passed; adds forward Delete. Original failed logs and reports are retained unchanged.

No competitor application was executed, no current input source was changed, and no fake conversion results were used. Offline installed first activation, external editor compatibility, latency, universal binary behavior, real OS-kept-key lifecycle remain separate gates.

- Attempt 05: 12 / 12 passed against the canonical packaged header and `lib/librime.1.dylib`, linked by full library path. No manually acquired dist path is needed by the runner. Build-only private headers come from the canonical staging helper SDK; runner never downloads or deploys schemas.

- Attempt 06: existing 12 cases passed; the new fault fixture failed its setup expectation. One-session next input queries call `ScriptTranslator::Query -> FinishSession`, flushing pending learning before the next final commit. The actual result correctly disabled typing and retained raw recovery text `n`; no false post-commit claim is made.
- Attempt 07: compilation failed during a concurrent production integrity-check edit (`id.pathComponents`); the root agent corrected the native source. Test did not execute.
- Attempt 08: **14 / 14 passed**, including actual immutable-data corruption rejection and a real encrypted-learning filesystem-failure commit.

## Final fault fixture and source timing

Source reviewed: pinned librime `src/rime/gear/script_translator.cc` (`Query` calls `FinishSession`), `src/rime/gear/memory.cc` (`OnCommit` calls `StartSession`), `src/rime/dict/user_dictionary.cc` (`NewTransaction` flushes a previous pending transaction; `UserDictionaryComponent` shares the same dictionary DB through its pool). Arion `EncryptedDb::CommitTransaction` merges pending updates then calls `persist`; `writeFile` creates an exclusive ciphertext temporary file, writes/fsyncs, renames and fsyncs the directory.

The verified fixture composes session A first, then session B commits actual 你好, leaving pending learning in their shared encrypted DB. Only then it renames the isolated vault directory and places a regular file at its old path. Session A's Space commits a real candidate while the shared pending transaction flush fails at the actual filesystem boundary. The host receives 你好 exactly once; typing reports unavailable and rejects later input while preserving committed text. The original directory path is restored in `@finally`. This fixture neither injects an error return nor calls the DB directly.

The integrity preflight copies real data and its original runtime manifest, changes `build/default.yaml` bytes without changing the manifest, and invokes the production engine startup. It gets `ArionKeyboard` error 2 with no user directory and no engine session, then the unchanged canonical data starts normally. No source registration or Keychain access occurs.

## Actual learned ranking, restart and cancellation

Attempt 10 passes **15 / 15** cases, preserving all previous 14 gates. For `shishi`, real initial candidates begin `[实施, 事实, 试试, 实时, ...]`. Selecting the actual second candidate 事实 commits it. After all sessions stop and the engine finalizes, an actual `.arion-userdb` encrypted file exists; its ciphertext contains neither the selected synthetic word nor raw `shishi` / `shi shi` canaries. A read-only instance of the same encrypted backend with the isolated synthetic key finds the real learned record. No synthetic learning record is inserted.

The test restarts actual librime with the same immutable data, ciphertext and key. Real candidates now begin `[事实, 实施, 试试, 实时, ...]`: the confirmed nonfirst word is promoted after restart. It then cancels both a raw composition and a provisional prefix selection, closes all sessions and compares every decrypted engine DB key/value and metadata tick against the completed-learning baseline. The complete map is unchanged, proving those cancellations did not add or increment learning. The test also confirms the old callback is rejected after shutdown.

Attempt 09 failed compilation in a third-party Boost unused-parameter warning. The runner now marks the canonical private SDK as a system header directory, keeping `-Wall -Wextra -Werror` for Arion code. Attempt 10 uses the unchanged production engine/typing/encryption implementations; only regression test and runner files were edited here.

## Declared schema scenarios

Attempt 11 passes **20 / 20** cases. Added unchanged real-engine fixtures: apostrophe `ni'hao` supplies and commits 你好; abbreviation `nh` supplies 你好 without asserting it must rank first; `lvse` preedit displays ü and supplies actual 绿色; `nihao Space ,` produces exactly 你好， with no raw Latin residue or newline; `arionxyz Return` commits exactly the literal raw input without a newline. All prior learning, corruption, editing and storage-failure gates still pass. These are bounded synthetic cases, not a general quality benchmark.


## Callback failure revocation and destruction

Attempt 12 passes **22 / 22** cases, preserving all attempt 11 gates. The production typing adapter now handles refresh failures centrally: actual committed text is inserted once, pending preedit is retained for explicit recovery, candidate data and revision are revoked, and the owned marked range is cleared. Both selection and highlight callbacks refuse unavailable sessions. A real filesystem fault in a shared encrypted userdb triggers this path for separate mouse-selection and mouse-highlight fixtures; neither stale nor current callbacks can insert text after revocation, and recovery text survives rejected callbacks.

The destruction fixture creates a separately owned real Rime session and verifies its presence with `RimeApi.find_session`. Releasing the Objective-C owner removes that session from the actual engine service while preserving the client mark-call, insertion and UI-callback counts. Cleanup is generation-checked and performs no client text writes during object destruction.

`typing-attempt-12-identities.json` binds all tested source/runtime bytes; `typing-attempt-12-sources/` preserves matching originals immediately after the run. Attempt 11 records and identities remain unchanged. These tests use synthetic keys and isolated temporary files, without Keychain access, source registration, selection, or installed IMK dispatch. This run does not validate OS host lifecycle acceptance.


## Translated x86_64 execution

Attempt 13 passes **22 / 22** cases with the unchanged production native sources, compiling a thin x86_64 test executable and explicitly launching it with `/usr/bin/arch -x86_64`. The physical host and Node runner are arm64; execution therefore uses the already installed Rosetta 2. The runner checks the compiled architecture using `lipo`, logs host/runner/executable architecture, and records this boundary in the result's `execution` object. No translation runtime was installed.

The reusable runner now accepts `--arch arm64|x86_64` as an architecture value, independently of path options, and always launches tests through the selected `/usr/bin/arch` slice. All prior 22 actual engine/AppKit cases pass. `typing-attempt-13-identities.json` and `typing-attempt-13-sources/` preserve exact matching bytes; attempt 12 remains untouched.

This is translated x86_64 user-space execution on Apple silicon. It does not establish physical Intel hardware acceptance, installed IMK event dispatch, real OS input-source lifecycle, or production Keychain access.
