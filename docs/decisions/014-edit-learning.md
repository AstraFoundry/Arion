# ADR 014: Learn vocabulary from the user's corrections of inserted text

## Status

Accepted (product decision). Amends MEM-02 of [003](../../specs/003-memory-sync.md), which had learning only propose records, and adds MEM-04/05. Extends PRV-02 of [005](../../specs/005-privacy.md) and CTX-06 of [009](../../specs/009-contextual-rewrite.md).

## Context

Typeless, Doubao IME and Chatterfly all watch the field after dictation, notice when the user fixes a word, keep the fix and use it again ([research](../research/edit-learning.md)). They differ in where each step runs:

- **Typeless** diffs on its server, adds the word to the account dictionary and shows it for 5 s with an undo.
- **Doubao** stores server-cleaned pairs in a local table, with use and recall counts, a 30-day expiry and suppression of reverted pairs. It feeds the top pairs to its on-device speech model.
- **Chatterfly** mines terms on its server and syncs them into a local dictionary.

Arion had approved vocabulary that a user could type in Settings, and every entry went into the rewrite prompt.

## Decision

**When a field is followed.** Learning is on by default, with a switch in Settings › Vocabulary. While it is on, a field Arion committed into stays followed if all of these hold:

- the field is readable;
- it is not secure and its app is not on the password-manager denylist;
- the session did not go through an IME composition.

**Reading and settling.** `application.EditTracker` reads a bounded excerpt around the cursor every 0.4 s. Each read has a 250 ms budget, and three failed reads abandon the field. The edit settles at whichever comes first:

- 15 s without change;
- focus leaves the field;
- the field is cleared, which happens when a chat message is sent; the last text read is then used;
- the next dictation starts;
- 60 s pass.

**Extraction** (`domain.ExtractCorrections`). Every change must lie inside the inserted text, with the text around it intact. The insertion and the edited text are compared word by word for Latin and character by character for CJK. For CJK changes, the local word-segmentation port supplies boundaries for both the inserted and corrected text. The domain retains complete words on both sides and merges changes within a word before projecting unchanged edges. Each pair is limited to 16 characters per side and three changes per edit. A standalone one-character word is learned only with an adjacent complete question particle (for example 爆吗 → 报吗); other ambiguous boundaries are skipped. No arbitrary neighboring character is appended. macOS uses Core Foundation via the existing purego runtime; missing/invalid segmentation yields the content-free `words_unavailable` outcome, with no guessed pair. Nothing is learned from:

- typing before or after the insertion;
- deletions or insertions alone;
- punctuation;
- changed numbers or times;
- sentence-start capitalization, and any change that only lowers letter case;
- a single Latin letter as the heard form;
- changes between characters chosen by grammar or meaning (他/她/它, 的/地/得, 在/再 and similar groups), like Doubao's exempt-word dictionary;
- edits that remove more than two thirds of the insertion or add more than twice its size.

**Approval.** A learned pair applies at once. The pill shows "Learned *from* → *to*" with an undo for 5 s. The overlay takes clicks only in a pill-sized strip and only for that time. Undo, or correcting the learned word back, removes the entry.

**Memory** (`domain.Learn`, `PruneMemory`):

- Learning the same pair again counts as a use.
- A learned entry may be re-pointed to a new correction. An entry the user typed is never changed by learning.
- Learned entries expire after 90 days unused. Beyond 2,000 entries, the least used learned entries are evicted.
- Everything lives in the existing AES-256-GCM vocabulary file. Each write is atomic and fsynced, and a file that fails authentication is never overwritten.

**Recall.**

- `domain.Vocabulary` indexes the approved entries once per change and replaces them in one left-to-right pass: whole words, longest match first, no chaining. This applies with or without polishing. Each entry that fires counts a recall.
- With polishing on, only the entries whose heard or written form occurs in the text reach the prompt's terms, at most eight.

**Logs** carry only the settlement, the verdict and counts (PRV-05).

## Consequences

- Unlike all three competitors, the diff, the memory and the recall stay on the device, and nothing is uploaded.
- The prompt no longer grows with the vocabulary.
- Replacement no longer slows with the number of entries.
- Field text read for learning is held only in the tracker's memory and dropped when it settles.
- Word boundaries are linguistic evidence, not a guarantee for unknown names or compounds. Ambiguous edits may teach nothing (for example 全免 → 全棉 when the OS returns single characters); users can still add explicit pairs. Existing stored fragments are not rewritten automatically because their original sentence context was not retained.
- A pair applies after a single correction, so a mistaken edit is applied until it is undone or corrected back. Content changes that look like misrecognitions, such as one weekday replaced by another, are learned. The undo, the Settings list and reversal handling are the safeguards.
- Edits Arion cannot see are not learned: Windows (no field reading yet), IME compositions, hosts whose Accessibility value is unreadable, and edits after the field lost focus.
- A user who rewrites polished text back to their own wording teaches that wording, as with Typeless.
- Historical extraction/recall results before the word-boundary change, measured in [edit-learning-v1](../../benchmarks/results/edit-learning-v1/README.md):
  - On labelled edits, pair precision is 0.95 and recall 0.91, with nothing learned from 23 non-corrections. That set is a development set.
  - Held out, memory from dev corrections changed none of 150 test dictations: no benefit and no harm, because the errors did not recur.
  - When the same sentences are re-dictated, errors drop from 8.5 % to 3.6 % (mixed technical speech 23.9 % → 7.3 %), with 1 of 114 worsened.
  - With 2,000 entries, recall p99 is 18 µs, against 687 µs for the loop it replaced.
  - Learned writes take about 10–13 ms, off the main thread, and every injected storage fault left dictation working.

## Alternatives considered

- **Propose, then approve in Settings** (MEM-02 as first written). It is the most conservative, but it gives no automatic recall, and no competitor does it.
- **Apply after the second identical correction** (Doubao-style counting). It means fewer mistaken entries but slower recall. The user chose Typeless's one-shot learning with undo.
- **Recall only through the rewrite prompt** (Doubao). That does nothing without polishing, and the shipped 1.7B model follows terms unreliably.
- **Server-side diffing** (Typeless, Chatterfly). Rejected by PRV-01.

## References

- [Learning from corrections](../research/edit-learning.md) and its [ledger](../research/evidence/edit-learning.json).
- [003 Memory](../../specs/003-memory-sync.md), [005 Privacy](../../specs/005-privacy.md), [009 Rewrite](../../specs/009-contextual-rewrite.md).
