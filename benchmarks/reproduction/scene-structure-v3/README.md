# Proportional structural editing v3

This development diagnostic turns two user-selected successful v2 outputs into
explicit preferences: group each owner with their task and date; separate an
email's spoken salutation from its natural short body. Prefer original wording,
preserve request/completion intent and do not force headings, formal stock
phrases, a separate closing or bullets for a short single item.

`work-zh.json` and `email-zh.json` are experimental templates, not production
changes. The two selected examples are included in demonstrations and labeled
`user-selected-anchor`; reproducing them is not generalization evidence. Six
additional authored development probes test related boundaries. Both v2 and v3
run on the same isolated worker/model. All 16 outputs are retained with explicit
assistant review under [raw evidence](../../runs/scene-structure-v3/).

The completed local development review found v3 retained meaning in all eight
cases and produced the requested proportional formatting. V2 changed the
first-person later-reply clause in one uncertain-attendance case and had weaker
owner grouping/short-message formatting. This small exposed diagnostic does
not establish release quality or OCR correction. See the separate
[context experiment](../context-repair-v1/README.md).

From the repository root, `pnpm rewrite:context:structure --out <new-directory>`
replays inference after preparing the pinned v1 build/runtime/model described
in [the v1 workflow](../scene-rewrite-alignment-v1/README.md). `freeze.json`
records the corpus, templates, runner and protocol before inference. Do not
refreeze published inputs or overwrite a run. The report/verification workflow
is shared with the context experiment.
