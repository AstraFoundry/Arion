# Packaging evidence archive

54 development and 96 evaluation generations, complete source-bound assistant
reviews, the pre-evaluation selection and common overlay sources are retained.
`archive.json` pins original bytes. Binary/runtime/weights stay in the ignored
workspace and are identified by recorded hashes, not bundled in this archive.
The report is deterministic and replays without model execution. See the
[reproduction instructions](../../reproduction/rewrite-packaging-v1/README.md).
