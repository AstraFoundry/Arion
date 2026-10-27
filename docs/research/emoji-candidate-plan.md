# Emoji candidate implementation proposal

Status: The native filter, typed menu integration and macOS preference described below are implemented. The index is generated offline from checked-in, digest-pinned CLDR 48.2 Chinese annotations and Unicode Emoji 15.0 fully-qualified sequences, with 30 reviewed overrides; it has 1,740 exact triggers, 719 of which have exact entries in the pinned Pinyin dictionary. Dictionary presence alone does not prove reachability in the filter's bounded candidate window. Isolated real-Rime/AppKit tests passed exact `xiaolian` → `😊` commit, representative CLDR/flag/alias reachability, number/Space/punctuation selection, caret suppression/restoration, text-order preservation when disabled, and no `笑脸`/Emoji entry in encrypted Chinese learning records. Full OS rendering/installed-source acceptance and insertion-policy comparison remain open; the three-text/two-Emoji policy is provisional rather than measured optimal.

## Recommendation

Use a small Arion-owned native Rime filter, a typed candidate bridge and a deterministic menu policy. Build the Emoji data from pinned Unicode/CLDR sources and reviewed Arion aliases. Keep the existing Rime parser, encrypted Chinese learning, native candidate windows and engine commit route.

This adds explicit Emoji semantics where the current string-only menu cannot express them. It does not require a new Pinyin engine, generative model, Lua runtime, network service or a fork of librime. The native extension uses the same pinned Rime C++ ABI boundary already required by encrypted learning; its actual exports and packaging must pass the first integration gate below.

The first release targets macOS Simplified Chinese Pinyin. Put the implementation in the shared keyboard core and maintain Windows build compatibility, but do not claim Windows feature support before native TSF acceptance. English names remain available as metadata for accessibility; English-mode keyword completion, shortcode search, a separate picker and voice-to-Emoji substitution are subsequent scope proposals.

## Evidence and alternatives

| Option | Useful property | Limitation for Arion | Recommendation |
| --- | --- | --- | --- |
| Stock OpenCC Emoji filter | Existing engine-native candidate expansion with little integration code | General text conversion, shadow-candidate learning, and no typed host metadata | Use as a bounded comparison implementation, not the selected production design |
| Arion native Rime filter | Controls matching, candidate type, input range and learning behavior while retaining engine commits | Additional pinned C++ extension and metadata bridge to maintain | Recommended, subject to the integration gate |
| Host-only appended Emoji | Easy rendering and independent UI ordering | Introduces a second composition/commit owner or requires selecting a different engine candidate and substituting its result | Reject for this feature |
| Direct Emoji Pinyin dictionary | Real engine candidates with explicit spellings | Duplicates pronunciation/alias maintenance and can compete with ordinary word decoding | Consider only for a measured trigger-reachability gap |

Reviewed upstream sources:

- [rime-emoji configuration](https://github.com/rime/rime-emoji/blob/master/emoji_suggestion.yaml) and [mapping data](https://github.com/rime/rime-emoji/blob/master/opencc/emoji_word.txt) demonstrate original-word retention and engine-native expansion. The project documents traditional-character assumptions; its mappings are not automatically suitable for Arion's Simplified Chinese dictionary.
- [Rime 1.17 simplifier](https://github.com/rime/librime/blob/1.17.0/src/rime/gear/simplifier.cc) emits `ShadowCandidate` and can perform general text conversion after exact-word conversion fails. [Candidate unwrapping](https://github.com/rime/librime/blob/1.17.0/src/rime/candidate.cc) and [script-translator learning](https://github.com/rime/librime/blob/1.17.0/src/rime/gear/script_translator.cc) show why selecting a transformed candidate can be interpreted through its original Chinese phrase. That behavior is useful for script conversion, but should not silently define Emoji learning.
- [Rime candidate types](https://github.com/rime/librime/blob/1.17.0/src/rime/candidate.h) support typed candidates with an input range, commit text and preedit. This is a design basis, not proof that the proposed extension is already supported by Arion's packaged binaries.
- [rime-ice filter ordering](https://github.com/iDvel/rime-ice/blob/main/rime_ice.schema.yaml) and [conversion chain](https://github.com/iDvel/rime-ice/blob/main/opencc/emoji.json) demonstrate ordering constraints and why importing a whole configuration also imports unrelated symbol behavior.
- [Fcitx candidate construction](https://github.com/fcitx/fcitx5-chinese-addons/blob/master/im/pinyin/pinyin.cpp) and [symbol selection](https://github.com/fcitx/fcitx5-chinese-addons/blob/master/im/pinyin/pinyincandidate.cpp) demonstrate preserving the associated text and consumed Pinyin length.
- [Mozc Emoji insertion](https://github.com/google/mozc/blob/master/src/rewriter/emoji_rewriter.cc) and [environment filtering](https://github.com/google/mozc/blob/master/src/rewriter/environmental_filter_rewriter.cc) demonstrate deliberate insertion positions and client character capabilities.
- [CLDR names and keywords](https://cldr.unicode.org/translation/characters/short-names-and-keywords) supply locale data; [Unicode Emoji](https://www.unicode.org/reports/tr51/) defines sequences and presentation behavior.

These are source observations. No comparative runtime or license-compatibility conclusion follows from them. Any imported file needs its own provenance and license record; Arion's proposed dataset does not copy another input method's mapping wholesale.

## Proposed product behavior

Emoji suggestions are available offline in Chinese Pinyin mode, with a proposed default-on switch in Settings > Keyboard. The input method reads the app-owned preference on activation, consistent with existing keyboard preferences; an absent field has the documented default. Do not duplicate persistence in the host-owned punctuation options file.

Only exact matches of an eligible candidate word trigger suggestions. The word must consume the entire currently unselected Pinyin suffix, and the input caret must be at its end. Previously selected text is retained. Completion-only candidates, substring replacements inside generated sentences, punctuation, reverse lookup, English mode and voice composition do not trigger this release's suggestions. A reviewed exact multiword alias may trigger; arbitrary sentences do not.

For example, an eligible full candidate `笑脸` can expose Emoji alternatives; an arbitrary sentence containing those characters cannot be partially rewritten by the feature. This is an example of the intended contract, not an observed engine result.

Use neutral/default variants unless the trigger explicitly names a variant. Do not infer skin tone or gender from context. Keep flags and ZWJ sequences intact. Do not enumerate every skin-tone variant in the ordinary candidate bar.

Selecting a suggestion submits its exact engine candidate. Space, number keys, mouse, navigation, punctuation, cancellation and focus changes retain the existing engine ownership rules. User navigation freezes automatic selection for that composition revision.

## Candidate generation and identity

Register one shared `arion_emoji` filter before sessions open. It passes original candidates through unchanged and adds `arion_emoji` candidates after eligible words. Use a dedicated candidate derived from `SimpleCandidate`, rather than a Chinese `Phrase` or `ShadowCandidate` that unwraps to one. Copy the source input range and required preedit, while keeping source association as descriptive metadata rather than inherited learning identity.

Add a versioned C bridge that copies typed records from the same Rime session/menu as the engine-index operations. Records need candidate kind, exact commit text, input start/end, source association, Emoji sequence ID and engine index. Pair them with the host's session/generation/revision. An engine index alone is not a durable identity; a sequence ID alone is not a selectable candidate identity. Deduplicate only equivalent text for the same consumed range, preserving deterministic source association.

Do not encode candidate metadata in visible comments, infer kind from font/glyph appearance, borrow candidate pointers beyond the snapshot, or select an engine word and replace its commit string with an Emoji afterward. Capture the commit's Emoji provenance with the actual engine commit, before commit consumption; cancellation and session replacement clear associated metadata. A mixed Chinese/Emoji commit must not lose that provenance.

Native integration must establish that a dedicated non-Phrase candidate consumes the intended range, preserves preedit, commits correctly and avoids Chinese learning. If it cannot, record the failure and revise this design before production integration; do not silently substitute a shadow candidate or host-side insertion.

## Ordering

Keep the context ranker a permutation of text candidates. `menu.c` currently fetches a leading window of 27 entries and `ranking.cpp` uses displayed grapheme count as a proxy for full-input coverage. Mixing Emoji into that untyped window would change both the text ranking population and length semantics.

The updated menu gathers the same leading 27 text candidates, ranks that text set using the existing policy, then merges actual engine-produced Emoji candidates. Emoji never receives the Chinese ranker's word-length score. Carry input coverage as metadata; any change from the legacy text-length rule to range-based Chinese ranking needs a separate regression comparison, not an incidental change in this feature.

Use bounded expansion: at most one Emoji per eligible source among that leading text window, hence at most 27 additional engine candidates in that window. Preserve lazy loading for later text candidates and do not generate additional Emoji for the unbounded tail. The metadata bridge and menu must account for actual engine indices when skipping non-text entries. No candidate enumeration may exhaust the complete dictionary.

Provisional display policy: protect the first three text slots, insert at most two Emoji on the first page, and retain relative text order. Select Emoji priority by associated text rank, reviewed alias priority, then pinned data order. Additional generated Emoji remain reachable on later pages under a bounded merge policy; no hidden items may consume number-key slots. The exact later-page layout must be specified in the policy fixture before implementation.

Compare insertion after one, three and five text candidates using the same generated candidate set. Finalize the protected slots and per-page quota on development fixtures before held-out evaluation. Three/two is an initial proposal, not a measured optimum. A missing match creates no placeholder and does not change text order.

## Data and compatibility

The generated index is compiled into the native host and records Unicode sequence IDs. It ships no image/font assets or rime-emoji mapping. Source XML (base and derived annotations), Emoji test data, generator and deterministic header are checked in; `emoji_sources.json` records versions, hashes, source URLs and the Unicode-3.0 license. Automatic triggers are two to six Han characters, prefer exact CLDR short names over keywords, accept only closely formed keywords, omit keyword-only terms with more than two associated Emoji and produce at most one Emoji; the reviewed override list preserves chosen common-word behavior and permits deliberate one-character triggers. A small exclusion set suppresses ordinary words with misleading CLDR symbol names, such as `注册` → `®️`. Skin-tone variants are excluded from automatic suggestions. Never strip ZWJ or variation selectors from commit text as a normalization shortcut.

CLDR keyword coverage is broader than useful automatic suggestions. Keep reviewed overrides separately from raw annotations, with exact-match semantics and a reviewed single-character allowlist. Only eligible short names and sufficiently unambiguous keywords become automatic triggers; Unicode ordering is not treated as popularity.

Report trigger reachability against the actual pinned Pinyin dictionary: which reviewed triggers produce an eligible candidate within the bounded window, which do not, and why. Fix curated alias gaps explicitly. A pronunciation dictionary extension is justified only by a measured gap, and requires separately reviewed static data; do not assume every CLDR label is already typeable.

Declare a tested sequence subset for each supported OS/font baseline. CoreText shaping can assist diagnostics but a glyph-presence check alone does not prove a complex sequence renders correctly. Until newer sequences pass that matrix, use the verified common subset. Record excluded sequences; do not replace unsupported ones with different meanings. macOS uses its installed Apple Emoji rendering; bundle Unicode text and metadata, not Apple's image/font assets. Windows gets its own rendering matrix before release.

Provision and verify data through the existing immutable runtime manifest. Load lookup data at activation, with no deployment, download or disk reads per key. Updates follow the existing signed keyboard payload transaction; no live dictionary download is introduced.

## Learning, failure and privacy

The first release has deterministic Emoji order and no Emoji preference store. Choosing an Emoji must not boost the source Chinese word, learn the Emoji as a Chinese word, or train ASR/correction vocabulary. Legitimately selected Chinese segments retain their existing learning behavior. The host's `containsHan` commit gate is insufficient for mixed commits: commit provenance must prevent mixed Emoji content entering Chinese context learning, without undoing independent confirmed Chinese selections.

Later Emoji preference learning would require an independent encrypted record namespace keyed by trigger and sequence, final-confirmation semantics, reset behavior and its own tests. It is not implemented merely by reusing Chinese learning counts.

Matching uses the current engine candidates, not editor contents or screen context. No network calls, raw key history, candidate text or selected Emoji enter diagnostics. Benchmarks use authored, nonpersonal fixtures under the evaluation archive rules.

Disabled is an ordinary feature state. Missing/incompatible optional capability is an explicit unavailable state with localized status; it cannot silently advertise enabled suggestions. Immutable payload corruption retains the existing manifest-verification failure policy. Invalid metadata, stale revision or identity mismatch rejects the affected candidate interaction, preserves explicit recovery and never inserts guessed text. Do not bypass an error through clipboard delivery or a second commit route.

## Implementation sequence and ownership

| Priority | Work | Depends on | Completion gate |
| --- | --- | --- | --- |
| P0 | Minimal native integration probe using authored mappings | Pinned runtime/SDK | Real Rime candidate, full suffix consumption, no unintended learning, exact commit and clean cancellation; verify metadata and symbols on macOS and build on Windows |
| P0 | Proposed spec/decision updates and corpus policy | Successful probe | Add explicit Emoji acceptance cases to 007, extend ADR 017 and settle merge fixtures; keep existing IDs and record baseline text order |
| P0 | Deterministic data builder and provenance manifest | Approved matching contract | Reproducible bytes, valid sequences, notices, aliases, reachability and supported-sequence reports |
| P0 | Shared producer, typed bridge and menu integration | Probe, data | Bounded enumeration, unchanged text-only ranking population/order, stable IDs and real engine selection |
| P0 | macOS settings, renderer and commit provenance | Shared integration | Native lifecycle, accessibility names, light/dark layouts, page sizes and learning isolation |
| P0 | Paired policy evaluation and actual installed-source acceptance | Integrated macOS host | Frozen policy chosen on development fixtures, held-out report and real editor evidence |
| P1 | Windows TSF release acceptance | Shared integration, Windows render matrix | Native candidate visibility, edit-session ownership, selection, cancellation and encrypted-learning checks on Windows |
| P2 | Emoji preference learning or explicit English/shortcode search | Measured user need | Separate product contract, encrypted data/reset and evidence gates |

Implementation entry points are `ime/shared` for the producer/metadata/menu; `ime/shared/keyboard-data` and `scripts/keyboard-runtime.json` for immutable data; `scripts/stage-keyboard-runtime.mjs` and platform build scripts for packaging; `ime/macos/keyboard.m`, `typing.m` and `candidate_panel.m` for native integration; `ui/features/keyboard`, the existing application-owned preferences and `ui/shared/copy/locales` for controls/copy. The desktop shell remains MyGo and the portable Go voice core gains no Rime dependency.

## Acceptance and evidence

Retain IME-05, KBD-03/04/05/06, RNK-01/02/03 and their existing acceptance cases. Add proposed Emoji cases for:

- Exact triggers, reviewed aliases, unreachable triggers, no-match terms, single-character ambiguity, sentences containing a trigger and completion-only candidates.
- Full suffix selection after an already selected Chinese prefix; mid-input caret suppression; exact preedit/commit and preserved neighboring Emoji/CJK text.
- All supported page sizes, expanded rows, mouse/numbers/Space/arrows/punctuation, revision-scoped identity, changing settings and freeze after navigation.
- Cancellation, focus/mode change, voice takeover, missing capability, invalid metadata and unchanged Chinese learning after Emoji selections, mixed commits, restart and cancellation.
- Default/skin-tone sequences, flags, ZWJ families, presentation selectors, localized accessibility, dark/light appearance and the declared OS rendering matrix.
- Feature-off text candidates/order/learning matching baseline; feature-on text relative order and full 27-text ranking population matching baseline for identical engine learning state.

Report Emoji recall@page, rank/MRR, text displacement and simulated selection cost separately. Use common, ambiguous and non-Emoji prose/technical fixtures. Do not compare a changed candidate set as a ranker-only ablation. Freeze development and held-out cases separately and retain per-case real outputs, failures, provenance and hashes under `benchmarks/` per [008](../../specs/008-evaluation.md).

Measure added lookup/menu time and total key-to-visible-candidate p50/p95/p99, cold activation, enumeration bounds and RSS on the real minimum supported macOS baseline and a current supported release. A provisional engineering objective is warm added processing p95 below 1 ms and p99 below 2 ms on the declared reference machine; establish the baseline and freeze a justified gate before held-out runs. These figures are targets, not results. Engine tests cannot establish visible native latency or installed-source acceptance.

Run `pnpm check`, `pnpm keyboard:check` and `node scripts/test-shared-pinyin.mjs` as applicable after implementation; extend the existing native engine/typing/learning gates with production Emoji cases. Windows additionally runs its actual native Pinyin/TSF checks on Windows. Installed-source editor scenarios need separate owned AppKit, browser textarea/contenteditable and rich-text observations on macOS. Record screenshots for settings and every materially different candidate layout/locale. Missing OS capabilities remain not run, never passing substitutes.
