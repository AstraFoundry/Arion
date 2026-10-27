#ifndef ARION_SHARED_EMOJI_H
#define ARION_SHARED_EMOJI_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ArionEmojiCandidateMetadata {
    size_t start;
    size_t end;
    int emoji;
    char source[64];
    char sequence_id[32];
} ArionEmojiCandidateMetadata;

void arion_emoji_register(void);
/* Configure once before creating sessions. The host may reject sequences its
 * system font cannot shape as one Emoji; NULL accepts the pinned Unicode set. */
typedef int (*ArionEmojiSupported)(void *context, const char *text);
void arion_emoji_set_support(ArionEmojiSupported supported, void *context);
int arion_emoji_candidate_metadata(uintptr_t session, size_t index, ArionEmojiCandidateMetadata *out);
int arion_emoji_refresh_after_caret_move(uintptr_t session);

#ifdef __cplusplus
}
#endif
#endif
