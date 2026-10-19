# Doubao keyboard selection and host commit

## Scope and result

The retained macOS **DoubaoIme 1.0.2** has a concrete static path from its ordinary candidate window into client `insertText:replacementRange:`. The keyboard selection caller explicitly passes `isASR = false` to a shared insertion routine. This closes the earlier [keyboard report](keyboard-ime.md)'s missing keyboard-to-host insertion surface, without treating a speech insertion path as keyboard evidence.

The [evidence ledger](evidence/doubao-keyboard-commit.json) retains exact installed universal/arm64 identities, source intervals, complete interval hashes, selected instruction anchors, static metadata bytes and limits. Its eight intervals cover 24,168 bytes; 6,040 instruction lines were checked against original arm64 bytes. Two undecoded slots, `0x1c6760` and `0x1c679c`, remain explicit. No app was executed and no private profile, dictionary or traffic was read.

These are **ordinary LLVM/static metadata observations**, not new REA decompilation or runtime evidence. Prior REA/package identities and retained disassembly were reused. The stripped main lacks `LC_FUNCTION_STARTS`; compact-unwind records merge adjacent functions, so refined Swift intervals use direct-called prologues and observed subsequent entries. The exported native commit interval also contains two local helpers. Complete source-function recovery is not claimed.

## Concrete call chain

| Step | Source-bound observation |
| --- | --- |
| Candidate window | `candidatesWindow(_:didSelectCandidateRawIndex:)`, entry `0x1001a240c`, directly calls keyboard `selectCandidate(index:)` at `0x100189308` from `0x1001a252c`. |
| Swift engine selection | At `0x1001894c8`, selection invokes `OimeEngine.shared` metadata slot `+0x240`. Shipped metadata bytes bind that slot to exported `commit(absoluteIndex:)` at framework address `0x24dcc`. |
| Relative candidate index | The wrapper divides by page size, switches page and computes the relative index, then sends `-[ImeEngine commit:]` at `0x24f78`. The returned `CommitResult` carries a selected candidate and optional `convertRemain` separately. |
| Native Pinyin commit | `-[ImeEngine commit:]`, entry `0x82e0`, invokes native virtual slot `+0x58` at `0x8314`. The shipped Pinyin engine vtable binds that slot to `oime_engine::pinyin::Engine::Commit(unsigned int)` at `0x1c5e2c`. Runtime selection of that concrete receiver was not observed. |
| Full host commit | Selection concatenates actual candidate `commitWord` strings and directly calls `0x10018dbac` at `0x100189de4`, with `w3 = 0` for `isASR`. The shared routine reaches client `insertText:replacementRange:` at `0x10018eaf4`. |

The last API's selector was independently decoded from the original pointer/cstring bytes at `0x101324fa0`. This avoids trusting every LLVM annotation: the retained annotator sometimes repeats stale selector comments after unrelated Objective-C calls. The shipped Swift/native vtable pointers likewise have exact raw-byte records, rather than inferred targets from names alone.

## Partial composition and ownership

After engine selection, `0x100189758`–`0x100189788` appends the selected candidate object to a controller-owned array. Exported framework field-offset bytes establish `Candidate.commitWord = 0x58`, distinct from `word = 0x48`, `convertString = 0x28` and `composingString = 0x38`. The concatenation instructions read `+0x58`, establishing which candidate text is joined.

The suffix branch retains these selected objects, writes the returned remainder into controller input state at `0x10018a184`–`0x10018a190`, and calls the input-property observer `0x100136838` at `0x10018a19c`. The call passes the **old** raw string after storing the new suffix; it is not by itself a new decoder request. It also updates surrounding-input context through `0x100196db4`, which sends `setInputProceeding:following:changeMode:`. That helper is not the Pinyin decoder. Input update reads engine candidates through shipped metadata slot `+0x230`, manages marked text/candidate presentation and schedules work with a weak controller capture.

Controller fields are referenced through shared offset cells, including `0x101341820` for the selected-object array and `0x101341af8` for input. These cells are zero in the shipped image and initialized at runtime; their addresses do not establish concrete instance offsets. The focused follow-up below establishes one queued UI callback; complete suffix-decoder re-entry and focus/session ownership remain untraced.

The full-selection branch joins `commitWord` values and uses the keyboard caller of the shared insertion routine. The routine reads a cached text client or the controller's `client`, handles replacement-range state and calls the actual insertion API. Its other ASR-specific paths do not invalidate this explicit keyboard caller. Static reachability does not establish the current runtime client, host acceptance or correctness across focus changes and reentrancy.


## Queued refresh follow-up

A source-bound follow-up validates six intervals, including the reused input observer: 7,808 bytes / 1,952 instructions. Five newly bound intervals cover 2,228 bytes / 557 instructions. The same artifact identities apply, and every focused anchor and interval hash was checked against original arm64 bytes. These counts are separate from the earlier eight intervals and are not added as independent source-function coverage.

At `0x100137c1c`, the observer captures **current** raw input from controller state. It weakly captures the controller at `0x100137c6c`, stores the weak box and string in a closure context, and submits it to `DispatchQueue.main.async` at `0x100137d78`. Thunk `0x1001aabd0` tailcalls callback `0x100137e04`. That callback returns if the owner is gone, current input is empty, or Swift string equality rejects the captured/current input match (`0x100137e3c`–`0x100137e88`). The matching path calls candidate UI refresh `0x100188a78` at `0x100137eb8`.

The refresh obtains actual candidate availability through shipped engine metadata slot `+0x230`, then reloads/shows or clears the window. The exported symbol at framework `0x24a54` is **`getCandidates(startIndex:count:)`**; the refresh passes start index `0`, count `1`. The second argument is a count, not a reset flag. This is a queued **UI refresh**, not an established asynchronous decoder call. The surrounding-context helper and the queued refresh must not be substituted for an untraced suffix-conversion entry.

The callback directly establishes weak lifetime and input-equality guards. It does not establish a session, focus or generation token. An unnamed integer increment in the observer (`0x100136b18`–`0x100136b38`) is not read by this callback and is not labeled as a generation guard. Empty-input paths also clear marked text and controller candidate arrays. Focus-loss entry, same-input re-entry, cancellation rollback and the remaining decoder path are unobserved; these limits establish neither a competitor defect nor a safety guarantee. Arion's session/generation guards remain an independent design.

## Learning and remaining limits

Selection snapshots `usrDictSaveGeneration` before entering the engine. Native Pinyin commit contains common-phrase/history commit and dictionary rollback-state operations. A controller counter compares against **50**, then calls `saveUsrDictIfNeededSinceGeneration:sync:` with `sync = false`; a false result enters explicit failure handling. This establishes deferred generation-aware persistence, not a disk write on every key.

Native candidate commit precedes final host insertion. Therefore this evidence does not justify assuming that only fully inserted text can change engine learning. Exact partial-selection learning predicates, cancellation rollback and persisted records remain unknown.

The window-to-selection and full keyboard-to-host insertion surfaces are now established statically. Full suffix re-decoding, focus/session ownership beyond the established weak/input-equality UI callback, runtime receiver selection, learning persistence and actual editor behavior remain partial or unobserved. No competitor code, dictionary or model is an Arion implementation dependency.
