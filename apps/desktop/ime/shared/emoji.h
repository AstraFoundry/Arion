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
int arion_emoji_candidate_metadata(uintptr_t session, size_t index, ArionEmojiCandidateMetadata *out);
int arion_emoji_refresh_after_caret_move(uintptr_t session);

#ifdef __cplusplus
}
#endif
#endif
