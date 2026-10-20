# Keyboard and voice vocabulary sharing

## Scope and conclusion

The inspected macOS clients have concrete cross-mode vocabulary/context paths. They do not establish that an entire keyboard base dictionary is copied into a speech model. Doubao exports selected personal typing words as speech hotwords. Chatterfly supplies per-application committed-input history to speech recognition and loads account speech terms into keyboard candidates.

Targets: installed Doubao IME **1.0.2**, and installed Chatterfly **1.0.8.13459**. The latter differs from the **1.0.6.13398** artifact in earlier reports; addresses and identities must not be mixed. The [ledger](evidence/keyboard-voice-lexicon.json) preserves hashes, addresses, byte-validation counts, REA Evidence IDs and coverage limits. No competitor application was launched, private user database read, or live traffic intercepted in this investigation.

| Question | Doubao | Chatterfly |
| --- | --- | --- |
| Can typing information reach speech? | Yes, statically: personal Pinyin user words are exported, filtered and attached as speech hotwords | Yes, statically: committed input is collected and per-app history enters ASR configuration under the enhanced-recognition gate |
| Is this a full base-dictionary import? | Not established; the recovered path selects user words and common phrases | Not established; the recovered speech input is history context |
| Can speech vocabulary help typing? | ASR NER records are searched by the Pinyin user-dictionary path | Account speech terms build a local voice user dictionary consumed by the typing candidate loader |
| Is every word guaranteed to improve the next recognition? | Unknown: active switches, filtering, model consumption and accuracy were not exercised | Unknown: active switches, service consumption and accuracy were not exercised |

## Doubao: user words become bounded speech hotwords

REA/Ghidra recovered three functions from the installed framework's arm64 slice:

- `Engine::AsrGetUserWord`, entry `0x1d046c`, Evidence `ev_f30ad536737c0529fdd53b8219bb92ac09764490c2a6436e9db3c814f990fffb`.
- `Engine::AsrGetGlobalContext`, entry `0x1d1328`, Evidence `ev_23f354667daceba72f870462e561495c1d4619ff948bcd6be98e4808abe2db18`.
- `ImeEngine::getASRUserWords`, entry `0xb8fc`, Evidence `ev_204fdd8f6abfe67ee5c9bcb7e688acaab2da6b7901f0adda60db18167a4b6d96`.

The user-word extractor obtains a user-side snapshot, calls `UsrDict::Export` and applies `AsrNgramScoreFilter`. It aggregates duplicate word records and consults common phrases. It produces a JSON `hotwords` array containing `word` and `class`; length checks include a cumulative UTF-16 word-text budget of 500 code units. This budget is not a count of 500 entries, and the complete scoring/filter policy remains unresolved. The separate global-context extractor serializes `hotwords.user_words` and `hotwords.common_words`, including richer word metadata.

The Objective-C++ wrapper dispatches task `0x26`. Original jump-table bytes bind that task to `AsrGetUserWord`; task `0x35` selects `AsrGetGlobalContext`. It parses the returned JSON and reads `hotwords`. This establishes a concrete bridge across the engine wrapper, rather than relying on function names alone. [DBL-user-words, DBL-global-context, DBL-wrapper, DBL-task]

In the main executable, context construction at `0x1000efe98` conditionally calls `getASRUserWords` at `0x1000f04e8`, creates `ASRContext.HotwordContext` objects, and invokes metadata slot `+0xc8` at `0x1000f0640`. The original metadata pointer binds that slot to `0x100b0cffc`; that setter addresses offset cell `0x101357f68`, named `ASRContext.hotwordsInfo`. Thus extracted typing words reach the actual speech-context object. A recognition setup caller invokes this context constructor at `0x1000f2660`. The guarding preference is not fully identified here, so unconditional/default enablement is not claimed. [DBL-asr-context, DBL-hotword-setter]

There is a separate upload surface: `tryUploadUserWords` calls `getASRGlobalContext:endTimestamp:`, reads `userWords` and `commonWords`, and prepares the context-upload path. Local hotword construction and account/server context must not be conflated. Actual uploads, account convergence and retention are unobserved. [DBL-upload; earlier cloud-sync report]

The reverse direction also has concrete local linkage: `AddNerResponse`/`AddModifyPairNerResponse` update `AsrUserNerDict`; `Engine::SearchUserDict` calls its `PySearchCb` at `0x1dff30`. `ConvertByUserDict` calls `SearchUserDict` from the Pinyin conversion path. Speech-derived entities can therefore participate in typing candidates without being merged into the keyboard base dictionary. Which responses are produced and accepted in an active session remains unknown. [DBL-ner-ingest, DBL-keyboard-ner, DBL-keyboard-user-call]

The existing [edit-learning report](edit-learning.md) additionally establishes correction-pair selection into the local speech model's `USER_MODIFY` prompt slot. That is a distinct memory channel; it must not be substituted for proof that the hotword list reaches every offline decoder.

## Chatterfly: input history and personal voice terms

The installed current version has two separate sharing paths:

1. A native committed-text callback appends text to `SGInputHistoryCollector`. It flushes through `SGInputHistoryStore.saveRecordWithApp:content:`. With enhanced recognition enabled, the speech path reads current-app history, fills `inputHistory`, and serializes it into the ASR configuration's `context_hist`. If its serialized UTF-8 JSON exceeds 6000 bytes, the field is removed. This is a concrete typing/history-to-speech path; it does not establish wholesale dictionary export or the service's internal entity extraction.
2. Account personal terms are synchronized through the logical `/lexicon/term/sync` route, parsed as voice-term records and compiled into `VoiceUsrDict`. The typing candidate loader calls `AddVoiceUsrEntry`. This establishes account voice-term reuse in typing candidates. The service alias is not a verified public hostname, and these static paths do not establish live synchronization.

The history channel can contain committed text from more than ordinary keyboard typing. It is not evidence that every stored record was manually typed. Default enablement, precise retention, all collection exclusions and remote handling remain separate questions. The earlier [cloud-sync](competitor-cloud-sync.md) and [edit-learning](edit-learning.md) reports describe account term CRUD and speech correction mining for their older selected version.

## Implications for Arion

The [practice and evaluation plan](keyboard-voice-lexicon-plan.md) maps these observations to proposed Arion ownership, privacy, implementation stages and measurable adoption gates. It does not change shipped behavior.

These are design implications, not implemented behavior or recovered competitor source:

- A shared personal-term layer can feed keyboard candidates and bounded ASR hotwords through different adapters. Engine-specific base dictionaries and ranking data can remain independent.
- Keep correct spellings, pronunciation, confirmed usage and correction pairs as distinct data. A typing-frequency record is not automatically an authorized speech correction.
- Retrieve and cap recognition context rather than inserting an entire dictionary. Evaluate recognition benefit and ordinary-word corruption separately.
- Treat temporary application history as a separate context capability with its own controls. Arion's current privacy contract does not authorize adding persistent typing history merely because a competitor has it.

## Coverage limits

The concrete client-side bridges above are answered statically. Current defaults, an end-to-end synthetic-word recognition test, server ranking/retention and cross-device convergence remain unresolved. No recognition-quality or latest-version claim follows from these artifacts. Ghidra's C++ pseudocode contains imperfect ABI/type recovery; original call instructions, jump-table bytes and metadata corroborate the bridge, while exact parameter names and all filter thresholds remain qualified.
