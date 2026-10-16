# Competitor cloud synchronization

This investigation separates account-backed synchronization from remote inference, correction mining, feedback and telemetry. It inspects Typeless **2.8.1 (2.8.1.130)**, Doubao IME **1.0.2 (1)** and the Chatterfly **1.0.6.13398** macOS payload. These are the selected artifacts, not a claim about the latest release on every platform.

The [ledger](evidence/competitor-cloud-sync.json) preserves artifact identities, reused REA Evidence and focused static observations. Ordinary source/disassembly observations have local identifiers; they are not new REA execution evidence. No application was launched, no private database or recording was read, and no service request or two-device synchronization experiment was performed. Server retention, deletion and encryption enforcement remain unverified.

## Comparison

| Data | Typeless | Doubao IME | Chatterfly |
| --- | --- | --- | --- |
| Personal vocabulary | Account-backed dictionary CRUD, including manual and automatically learned terms | Account dictionary synchronization; also separate recognition-context uploads | Account-backed personal-term CRUD; typing dictionary download/merge/upload; server-mined speech terms downloaded into a local voice dictionary |
| Preferences | Selected account language, translation, output-language and personalization controls | Account-sync controls and typed settings, fuzzy-sound and recent-punctuation surfaces; exact subset needs qualification below | General settings synchronization not established in the selected product branch |
| Dictation history | Optional history synchronization, including text and record metadata | Local history/diagnostic storage; no history synchronization established | Local history query/storage; no history synchronization established |
| Original audio | Excluded from the recovered history-push record; cloud inference is separate | No audio backup established; online recognition and optional feedback/improvement uploads are separate | No audio backup established; remote speech/text processing is separate |
| Corrections | Before/after field content sent for learning; learned dictionary entries are account-backed | Edits sent for extraction, returned pairs stored locally; cross-device pair-table synchronization unproven | Before/after speech text sent for term mining; term synchronization does not establish full edit-history synchronization |
| Clipboard | Context processing does not establish clipboard backup | Explicit Super Transfer is separate from local mobile clipboard history | Clipboard in rewrite context does not establish clipboard backup |

“No synchronization established” is a bounded finding, not proof that no implementation exists.

## Typeless

`TLS-01` and `TLS-02` recover an explicit history-push shape: record/user identifiers, platform, mode, status, timestamp, duration, app version, `refined_text`, `mode_meta`, deletion and concurrency metadata. **`mode_meta` can include selected source text and Ask AI results.** Audio bytes, local audio path and `audio_context` are absent from this shape. This exclusion says nothing about separate cloud recognition requests.

`TLS-11` establishes account-backed dictionary operations and bulk imports. `TLS-12`–`TLS-14` identify remote preferences: `personal_auto_style_on`, `user_info.locale`, translation target languages and dictation output-language mappings. Cloud history enablement and retention are separate controls. Actual refresh on another device was not observed.

`TLS-09` instead places shortcuts, microphone selection, startup, appearance, sounds, window visibility and local history retention in the local app-settings store. The focused pass found no general account upload for these settings.

The history-enabled initial value is unknown (`null`), then fetched remotely. The client supports disabling, deletion tombstones and a cloud wipe; accepted remote tombstones also delete corresponding local recordings. The vendor states that history synchronization is optional and disabling it deletes cloud history text. This is a server promise, not an observed erasure. [Data controls](https://www.typeless.com/data-controls), [privacy policy](https://www.typeless.com/privacy).

`TLS-10` separately submits inserted text, before/after field content, app and device context for personalization. The vendor describes retained abstract style patterns; their schema and cross-device reuse were not recovered. [Personalization](https://www.typeless.com/help/quickstart/personalization).

## Doubao IME

The official IME policy explicitly lists account profile information and personal dictionary data in cross-device synchronization. Its basic mode, cloud-processing mode, optional improvement program and feedback uploads have separate rules. Do not apply the general Doubao chatbot policy to this input method. [IME privacy policy, sections 1–2.7](https://lf3-cdn-tos.draftstatic.com/obj/ies-hotsoon-draft/wave_ime/ime_privacy_policy.html).

The installed package adds account-sync preferences for global synchronization, settings, fuzzy sounds and user dictionary data. Its typed handlers include settings, fuzzy-sound and recent-symbol synchronization. The recovered macOS settings mapper registers five entries:

| Local preference | Remote setting |
| --- | --- |
| `asrPunctuationPostProcessMode` | `audio.punctuation_display` |
| `enableASRMulticonnection` | `audio.other.allow_switch_mobile_wifi` |
| `isSpaceBetweenZhAndNumEnabled` | `keyboard.quick_fmt.numensym_before_space` |
| `inputHistoryUsable` | `privacy.smart_input.cloud_use_input_info` |
| `userDictUsable` | `privacy.smart_input.cloud_use_user_words` |

The multiconnection/mobile-Wi-Fi names are preserved without asserting a broader networking behavior. The much larger shared settings enum is not the macOS mapper's registration list. In particular, synchronizing `inputHistoryUsable` synchronizes a permission/preference, not the history's text contents.

Fuzzy sounds and recent English/Chinese punctuation have compiled sync handlers and native key-return code. Their active registration and runtime enablement were not independently established, so they remain supported synchronization surfaces rather than measured account data transfers.

The correction loop is separate: `EL-DB-upload` and `EL-DB-store` establish edit submission followed by local `modify_pair` storage. Uploading edits for extraction does not establish synchronization of that table or its usage/recall counts. See [edit learning](edit-learning.md).

Super Transfer advertises cross-device copy/paste. The policy's local-only mobile clipboard history is a different feature; neither statement should be generalized to all clipboard operations. [Official desktop product page](https://shurufa.doubao.com/pc).

## Chatterfly

The preference app's personal-term list/create/update/delete functions send authenticated requests. Their bodies carry term text, term identifiers or pagination as appropriate. This establishes cloud-backed manually maintained personal vocabulary, independently of the typing dictionary's bulk-file synchronization. The transport's logical service alias is not a literal localhost service or a demonstrated public hostname.

The preference app describes immediate dictionary synchronization and automatic synchronization every four hours. It also warns that clearing the local typing dictionary does not clear the cloud dictionary. These are shipped UI statements, not measured scheduling or deletion outcomes.

The bounded instruction path is more specific than inherited class names: `SGTimerTaskHub.syncAllUsrData` selects the Chatterfly/Yuanbao branch, whose `taskSyncYuanbaoUsrData` enters user-dictionary synchronization. The recovered dictionary task downloads, merges and uploads dictionary data. The generic Sogou settings/phrase/statistics implementations therefore do **not** establish that Chatterfly synchronizes those categories.

Speech vocabulary uses another path. `EL-CF-mining` submits recognized/final text and application context for term extraction; `EL-CF-sync` identifies server term synchronization into local `VoiceUsrDict`. This does not establish that complete before/after edits or history are backed up. The first-page history function queries a local store.

The packaged Tencent privacy-document links returned JavaScript shells without policy content in this investigation. No retention promise is inferred from them. Likewise, remote rewrite requests containing cursor text, clipboard and per-app history establish processing surfaces, not synchronized backups. See [context/rewrite evidence](chatterfly.md).

## Coverage and remaining questions

**Answered for the inspected code:** Typeless history record fields and selected cloud preferences; Doubao's five registered macOS settings mappings; Chatterfly account personal-term CRUD, typing-dictionary transfer and its narrower product branch; separation of correction-learning uploads from history synchronization.

**Partially answered:** full Doubao synchronization payload coverage, voice vocabulary synchronization consumers, cross-platform preference compatibility and actual default/active account settings.

**Unresolved:** observed two-device convergence, conflict outcomes, complete retention/deletion behavior, learned-style sharing and cryptographic key custody. None of these clients is established here to offer a server-blind encrypted memory relay.
