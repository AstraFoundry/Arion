#ifndef ARION_SHARED_MENU_H
#define ARION_SHARED_MENU_H
#include "pinyin.h"
#ifdef __cplusplus
extern "C" {
#endif

typedef struct ArionPinyinMenu {
    char **texts;
    size_t *indices;
    unsigned char *emoji;
    size_t count, fetched, page_size, page, highlight, top_row;
    int exhausted, expanded;
    int (*kind)(void *context, uintptr_t session, size_t index, unsigned char *emoji);
    void *kind_context;
} ArionPinyinMenu;
typedef int (*ArionMenuRank)(void *context, const char *const *texts, size_t count, size_t *order);

void arion_menu_dispose(ArionPinyinMenu *menu);
// Refresh candidates at page one, preserving expansion until explicitly
// collapsed or disposed at the end of the composition.
int arion_menu_build(ArionPinyinMenu *menu, RimeApi *api, uintptr_t session,
    size_t page_size, ArionMenuRank rank, void *context);
int arion_menu_build_typed(ArionPinyinMenu *menu, RimeApi *api, uintptr_t session,
    size_t page_size, ArionMenuRank rank, void *context,
    int (*kind)(void *, uintptr_t, size_t, unsigned char *), void *kind_context);
int arion_menu_load(ArionPinyinMenu *menu, RimeApi *api, uintptr_t session, size_t index);
int arion_menu_turn(ArionPinyinMenu *menu, RimeApi *api, uintptr_t session, int delta);
int arion_menu_move(ArionPinyinMenu *menu, int delta);
int arion_menu_grid(ArionPinyinMenu *menu, RimeApi *api, uintptr_t session, int expanded);
int arion_menu_prepare(ArionPinyinMenu *menu, RimeApi *api, uintptr_t session);
size_t arion_menu_visible_count(const ArionPinyinMenu *menu);
size_t arion_menu_row_count(const ArionPinyinMenu *menu);
int arion_menu_engine_index(const ArionPinyinMenu *menu, size_t index, size_t row, int grid, size_t *engine_index);

#ifdef __cplusplus
}
#endif
#endif
