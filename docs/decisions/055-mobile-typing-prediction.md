# ADR 055: Fuzzy Pinyin, phrase learning and prediction in the mobile keyboards

Status: Accepted for iOS. Amends [ADR 053](053-mobile-rime-typing.md).

## Context

The phone keyboard typed Pinyin without the desktop's fuzzy pairs, without context ranking and without next-word suggestions (联想). The product owner asked for these to follow the Rime community. Rime frontends (Trime, fcitx5-android, fcitx5-ios, Hamster) build fuzzy pairs from schema algebra, learn phrases through librime's own memory, and predict with the `librime-predict` plugin. The common Chinese data for that plugin is GPL-3.0; the Octagram grammar model is GPL code with data of 10–200 MB.

## Decision

- **Fuzzy Pinyin and typo correction.** The desktop's shared corrector (`spelling.cpp`), which mobile full-keyboard Pinyin already runs, with its eleven independent pairs and opt-in typo correction set per session (`arion_spelling_apply`), not a second set of schema substitutions. They are settings in the app, off by default; 9-key keeps exact spellings.
- **Phrase learning.** librime's `Memory` as is: parts chosen in one composition are saved as a phrase in the encrypted user dictionary. No second learner.
- **Context ranking.** The desktop's `ContextRanker` reorders the first page by the previous commit, with the desktop menu's eligibility (only candidates spanning the same input as the first), and learns pairs into the encrypted `arion_context` store. It never reads host text; a field change forgets the previous commit.
- **Prediction.** `librime-predict` (BSD-3-Clause, pinned in `packages/mobile-rime/runtime.json`) compiled into the bridge with the `rime-predict-zh` Simplified data (`predict-zhs.db`, GPL-3.0, about 0.5 MB), configured as that data documents. Suggestions follow each commit, a chosen one predicts again, and text the keyboard inserts itself (punctuation, space, return, delete) hides them. A setting turns prediction off.

## Consequences

- The app now distributes GPL-3.0 data. Its license and README ship in the keyboard's `licenses/rime-predict-zh`; the plugin's license as `licenses/librime-predict-LICENSE`. Distribution terms must be reviewed before release.
- No Octagram grammar: its size does not fit a keyboard extension's memory.
- The bridge test covers fuzzy pairs, prediction, dismissal, phrase learning across a restart and context-ranked selection.

## Alternatives considered

- **A license-clean continuation table** derived from the shipped `pinyin_simp` phrases and the user's own commits. Offered; the product owner chose the community data.
- **Fuzzy pairs as schema algebra.** Would duplicate the desktop's corrector and need a rebuilt prism per combination.
