# Context-dependent homophone corpus

`corpus.json` holds 50 original pairs in 20 homophone groups. Each pair is a previous word and a target word whose Pinyin is shared with other targets in its group, so the intended target depends on the previous word: 做 → 实验, 爱情 → 誓言 and 湖北 → 十堰 all type `shiyan`. Every pair appears four times in a seeded shuffled stream (seeds 2411–2413), as its context word followed by its target word.

The pairs were authored for this benchmark by the coding agent; no copied text, recordings or personal data were used ([provenance](provenance.json)). It is a small, exposed development diagnostic of context ranking, not a representative sample of user text. Its targets are deliberately ambiguous homophones, so the engine's first-occurrence top-1 rate is low by design and says nothing about general conversion quality, which the [quality corpus](../imk-quality-v1/README.md) covers.
