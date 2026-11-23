#ifndef ARION_ENGLISH_H
#define ARION_ENGLISH_H
#include <stddef.h>
#ifdef __cplusplus
extern "C" {
#endif
int arion_english_register(const char *data_directory, char *error, size_t capacity);
/* Call after finalizing Rime, before registering again in the same process. */
void arion_english_unregister(void);
/* For an English layout outside Pinyin: the lexicon's candidates for a typed
 * word (exact forms, completions, then other matches, in the typed case),
 * valid until the next call, and recording a chosen word so it completes. */
size_t arion_english_candidates(const char *word, size_t limit, const char **texts);
int arion_english_confirm(const char *word);
#ifdef __cplusplus
}
#endif
#endif
