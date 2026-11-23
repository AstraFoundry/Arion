// The phone keyboards' only door into librime (ADR 053). iOS calls it from
// Swift, Android through JNI. Both start the engine against data prebuilt on
// the host: nothing is deployed or downloaded on the phone.
#ifndef ARION_RIME_H
#define ARION_RIME_H
#include <stddef.h>
#include <stdint.h>
#ifdef __cplusplus
extern "C" {
#endif

// Authenticated encryption supplied by the OS (CryptoKit, Android Keystore).
// Each call writes a malloc'd buffer the bridge frees, and returns 1 on
// success; a failure must not fall back to plaintext.
typedef struct ArionRimeCipher {
    void *context;
    int (*seal)(void *context, const char *label, const uint8_t *plain, size_t plainSize,
                uint8_t **sealed, size_t *sealedSize);
    int (*open)(void *context, const char *label, const uint8_t *sealed, size_t sealedSize,
                uint8_t **plain, size_t *plainSize);
} ArionRimeCipher;

// Starts the engine once per process. `data` holds the shipped schemas with
// their prebuilt `build` directory, `user` is writable, and `vault` keeps the
// encrypted learning. With a null cipher typing works and nothing is learned.
// Returns 1, or 0 with a reason in `error`.
int arion_rime_start(const char *data, const char *user, const char *vault,
                     const ArionRimeCipher *cipher, char *error, size_t errorCapacity);
// Deletes the learning vault. Call before arion_rime_start or after stopping.
int arion_rime_reset_learning(const char *vault, char *error, size_t errorCapacity);
void arion_rime_stop(void);

typedef uintptr_t ArionRimeSession;
// Opens a composition on a schema (`arion_pinyin`, `arion_pinyin_9key`); 0 on failure.
ArionRimeSession arion_rime_open(const char *schema);
void arion_rime_close(ArionRimeSession session);

// A character key: letters, digits, the apostrophe syllable separator.
int arion_rime_type(ArionRimeSession session, char key);
// Deletes the last input key; 0 when there is no composition.
int arion_rime_backspace(ArionRimeSession session);
// Replaces the raw input, e.g. when a 9-key syllable is chosen.
int arion_rime_set_input(ArionRimeSession session, const char *input);
// Selects a candidate by its index in the whole list; learning happens here.
int arion_rime_select(ArionRimeSession session, size_t index);
// Commits the raw input as typed (the return key while composing).
int arion_rime_commit_raw(ArionRimeSession session);
void arion_rime_clear(ArionRimeSession session);

// The session's state after the last call. Strings stay valid until the next
// call on the same session. `commit` is the text to insert now, if any; it
// is handed out once.
typedef struct ArionRimeState {
    const char *input;
    const char *preedit;
    const char *commit;
    // Pinyin syllables of the unconverted input, for the 9-key syllable list.
    const char *const *syllables;
    size_t syllableCount;
} ArionRimeState;
int arion_rime_state(ArionRimeSession session, ArionRimeState *state);

// Up to `limit` candidates from `offset` in the whole list; returns how many
// were written. Strings stay valid until the next call on the same session.
size_t arion_rime_candidates(ArionRimeSession session, size_t offset, size_t limit,
                             const char **texts, const char **comments);

// The English layout, which composes nothing: the desktop's English lexicon's
// candidates for the word before the caret (exact forms, completions, then
// other matches, in the typed case). Strings stay valid until the next call.
size_t arion_rime_english(const char *word, size_t limit, const char **texts);
// Records an English word the user chose, so its own prefix completes later.
int arion_rime_english_confirm(const char *word);

#ifdef __cplusplus
}
#endif
#endif
