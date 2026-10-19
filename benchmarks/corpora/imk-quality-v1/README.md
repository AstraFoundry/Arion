# Original Pinyin quality diagnostic fixtures

`corpus.json` contains 100 newly authored synthetic Pinyin/target pairs: 45 common words, 35 sentences, 10 abbreviations, 5 v/ü forms and 5 coined compound targets. No copied text, recordings or personal data were used. This is an exposed development diagnostic, not a representative or independently annotated held-out population.

`provenance.json` records authorship, licensing, exact corpus/dictionary hashes, category counts and the excluded regression inputs. None of the 22 functional-regression inputs is reused as an exact fixture. Each input has one author-intended target; legitimate homophone readings can score as misses. The five coined complete target strings were absent as substrings of the pinned raw dictionary, but their constituent characters and sentence-generation paths may exist. They are a bounded novel-phrase probe, not proof of global OOV.

References and categories remain in the evaluator. Native requests contain only `id`, raw ASCII `input` and the two fixed unrelated warmup inputs when applicable. Every fixture/stratum/repeat uses a fresh process and empty synthetic-key encrypted profile, so committed evaluation targets cannot become memory for another fixture. Warmups are cancelled without committing and never scored.

See the [reproduction protocol](../../reproduction/imk-quality-v1/README.md). These disclosed fixtures cannot be relabeled as fresh unseen test cases in later development.

Corpus revision 2 corrects one extra `i` in `sentence-029`. The original frozen revision and its complete first run remain in [imk-quality-v1](../../runs/imk-quality-v1/README.md); the corrected full 600-process rerun is separate in [imk-quality-v2](../../runs/imk-quality-v2/README.md). No engine, driver or evaluation threshold changed.
