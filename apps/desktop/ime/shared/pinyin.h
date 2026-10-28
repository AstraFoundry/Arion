#ifndef ARION_SHARED_PINYIN_H
#define ARION_SHARED_PINYIN_H

#include <stddef.h>
#include <stdint.h>
#include <rime_api.h>

#ifdef __cplusplus
extern "C" {
#endif

/* The host owns runtime initialization, apartment affinity and persistent keys.
 * All text below is copied from one engine revision, never borrowed from Rime. */
typedef struct ArionPinyinSnapshot {
    char *preedit;
    char *commit;
    char **candidates;
    char **comments;
    size_t count;
    size_t cursor_utf16;
    size_t page;
    size_t page_size;
    size_t highlighted;
    int last_page;
    int ascii;
} ArionPinyinSnapshot;

int arion_pinyin_api_supported(RimeApi *api);
uintptr_t arion_pinyin_create_session(RimeApi *api);
void arion_pinyin_destroy_session(RimeApi *api, uintptr_t session);
void arion_pinyin_set_options(RimeApi *api, uintptr_t session, int ascii, int full_width, int english_punctuation);
int arion_pinyin_process_key(RimeApi *api, uintptr_t session, int key, int modifiers);
/* Commit through the schema's raw Return action without full-width conversion.
 * The caller must own a pending composition and consume the engine commit. */
int arion_pinyin_commit_raw(RimeApi *api, uintptr_t session);
int arion_pinyin_select(RimeApi *api, uintptr_t session, size_t index, int current_page);
int arion_pinyin_highlight(RimeApi *api, uintptr_t session, size_t index, int current_page);
void arion_pinyin_clear(RimeApi *api, uintptr_t session);
/* Returns 0 on success, 4 for an absent session, 5 for an engine read failure,
 * and 6 for invalid engine text/metadata or allocation failure. */
int arion_pinyin_snapshot(RimeApi *api, uintptr_t session, ArionPinyinSnapshot *out);
void arion_pinyin_snapshot_dispose(ArionPinyinSnapshot *snapshot);
int arion_pinyin_candidates(RimeApi *api, uintptr_t session, size_t start, size_t limit, char ***texts, size_t *count);
void arion_pinyin_candidates_dispose(char **texts, size_t count);
/* Strict UTF-8 boundary conversion, shared by both platform text adapters. */
int arion_pinyin_utf16_position(const char *text, size_t bytes, size_t *position);

#ifdef __cplusplus
}
#endif
#endif
