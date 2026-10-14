# Learning from the user's corrections

Do Typeless, Doubao IME and Chatterfly notice when the user fixes a word that was dictated into a field, store the fix as vocabulary, and use it next time? All three do. They differ in where each step runs. Evidence and byte offsets are in [edit-learning.json](evidence/edit-learning.json). All observations come from reading the shipped bytes; no app was launched and no request was sent.

| | Detect the edit | Store the fix | Recall it later |
| --- | --- | --- | --- |
| Typeless 2.8.1 | Client polls the field (`EL-TL-track`) | Server diffs; account dictionary (`EL-TL-report`) | Inferred server-side (`EL-TL-recall`) |
| Doubao IME 1.0.2 | Input method sees its own edits (`EL-DB-track`) | Server diffs; **local SQLite** `modify_pair` (`EL-DB-upload`, `EL-DB-store`) | **Local**: top pairs in the speech-LLM prompt (`EL-DB-recall`) |
| Chatterfly 1.0.6 | Client reads the field through Accessibility (`EL-CF-track`) | Server mines terms; synced into a local `VoiceUsrDict` (`EL-CF-mining`, `EL-CF-sync`) | Not established |

## Typeless

After an insertion the client watches the field. It starts only on an editable field whose content contains the inserted text, then polls the field and the keyboard. It settles when one of these happens:

- 15 s pass without an edit; each edit restarts the timer.
- Focus or the input box changes.
- The box is cleared.
- The user presses Enter.

A cleared box is judged by the last content the tracker read, which is how a sent chat message still counts. Large edits are dropped. Within the first 1000 characters, an edit is large when:

- less than 20 % of the text remains;
- more than half of it was removed; or
- the changes exceed twice its length.

The client posts the inserted text, the field before and after, and the app to `/user/traits`. The server's answer may name an `auto_add_dictionary` term. The floating bar then shows "added" for 5 s with an Undo button that deletes the entry. The diff runs on the server and the dictionary belongs to the account. No dictionary field travels with the recognition request, so any recall happens on the server.

## Doubao IME

As the input method, Doubao sees its own insert and delete events. It snapshots the field after each recognition commit and settles at three moments:

- before Return sends;
- on mouse activity;
- when the next recognition round begins.

Uploads go to `/api/v3/context/ime/modify_pair/v2`, unless the word diff is too large. The server returns cleaned "original → corrected" pairs, and the client writes them into a local SQLite table, `modify_pair`. Each row keeps:

- a use count and a recall count;
- timestamps;
- optionally, encrypted text with tags.

Pairs expire after 30 days by default and are evicted beyond a limit. When a pair and its reverse both exist (A→B, then B→A), the pair is never recalled.

`QueryModifyPairTopN` picks pairs in this order, up to a total limit:

1. today's pairs;
2. the most recalled pairs;
3. the most used pairs.

The selection fills the `USER_MODIFY` slot of the on-device speech model's prompt. When a corrected word appears in a result, its recall count increases. Doubao has the only complete loop, and the half that recalls runs on the device.

## Chatterfly

`SGVoiceEditTracker` reads an Accessibility baseline around the insertion, polls it and settles on triggers. For a signed-in user it posts the recognized text, the final text and the app to `/speech/mining/term`. The handler reads `new_terms[].text` and shows a "learned word" view on the speech bar. A timer later syncs `/lexicon/term/sync` into a local `VoiceUsrDict` in the Sogou kernel. Whether recognition reads that dictionary was not traced. A separate learner picks up the user's punctuation edits per app.

## What Arion takes

- Watching the field after insertion, settled by idle time, focus loss, a cleared field and the next dictation. All three products do this.
- Rejecting large rewrites, with Typeless-like size limits.
- Showing the learned pair right away with an undo (Typeless).
- Usage and recall counts, expiry, eviction and suppression of reverted pairs (Doubao).
- Recalling only what is relevant to the current text, within a bound (Doubao's top-N).

Unlike all three, Arion computes the diff, stores the memory and recalls it on the device, and nothing is uploaded. Doubao picks pairs before the text exists because its recognizer is the language model. Arion corrects after recognition, so it matches pairs against the text itself. [ADR 014](../decisions/014-edit-learning.md) records the decision and [edit-learning-v1](../../benchmarks/results/edit-learning-v1/README.md) measures it.

## Limitations

The defaults, consent gates, server-side diffing and the use of recalled terms were not observed at runtime. Typeless's recall is inferred from the absence of a dictionary field in the request. Chatterfly's consumer of `VoiceUsrDict` is unresolved.
