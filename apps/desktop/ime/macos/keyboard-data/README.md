# Arion Pinyin profile

`arion_pinyin.schema.yaml` explicitly adapts the pinned Apache-2.0 `pinyin_simp` schema. It retains that dictionary's full-Pinyin spelling rules, abbreviation, punctuation, native candidate selection and editing. The inherited `express_editor` makes Space choose a candidate, Enter commit raw input and Escape clear composition. Brackets and minus/equal page through nine candidates. Ordinary symbols use direct commit mappings instead of the inherited symbol lists, with paired Chinese quotes. Slash-code symbol recognition, stroke reverse lookup and alternate schemas are excluded from this profile.

The dictionary retains its AOSP PinyinIME provenance header. Prelude defaults/symbols are separately licensed LGPL-3.0 configuration assets. `scripts/keyboard-runtime.json` records immutable upstream URLs, exact bytes and licenses. The staging script ships those license and source references beside the compiled data.

`translator/db_class` requires Arion's encrypted user database. Never replace it with Rime's plaintext `userdb`. The runtime opens prebuilt data and must not deploy schemas or run Rime user-data synchronization. The immutable schema closure is compiled during staging, with no personal learning data. Build-only public/private headers and Boost dependencies stay in `.workspace/keyboard-runtime/sdk`.
