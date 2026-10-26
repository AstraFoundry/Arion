# Rime LLM reranker exploratory corpus

`corpus.json` contains 100 original assistant-authored synthetic fixtures, frozen before model inference. It is an exposed exploratory corpus, not independently reviewed or representative typing data. No personal text, external recordings or mined user histories are included. Fixture text follows the repository's AGPL-3.0-only licence.

| Group | Cases | Purpose |
| --- | ---: | --- |
| context | 48 | 24 fixed Pinyin inputs with two opposing semantic contexts |
| common | 12 | Preserve familiar words without preceding context |
| abbreviation | 8 | Initial-only inputs |
| long-input | 8 | Preserve the upstream ten-byte preedit gate |
| partial | 8 | Incomplete final syllable; intended prediction may not be offered |
| ambiguous | 4 | Multiple explicitly acceptable readings rather than invented unique intent |
| coined-name | 4 | Authored names; candidate-generation misses remain misses |
| truncated-context | 8 | Relevant clue intentionally outside the sixty-byte context tail |

`build_corpus.py` reconstructs the original fixture bytes into a new file; it does not derive references from model output. Both opposing-context cases share a linguistic group and are not independent semantic problems. The long-context group deliberately destroys information under the upstream tail policy and is reported separately.

Native collection receives only ID, raw input and unrelated cancelled warmups. It captures real production Rime first-page candidates and every intermediate key snapshot from an empty synthetic encrypted profile. The filter driver receives those candidate texts, preedit and authored committed context; references and acceptable lists remain in the scorer. Captured first-page candidates may consume different input spans. No artificial homophone candidate is inserted to improve oracle recall.

Replay uses [the study protocol](../../reproduction/rime-llm-reranker-v1/README.md). Publication makes these regression fixtures; they must not subsequently be presented as an unseen test set.
