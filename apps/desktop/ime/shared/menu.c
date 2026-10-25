#include "menu.h"
#include <stdlib.h>
#include <string.h>

enum { rank_window = 27, expanded_rows = 6 };
static size_t minimum(size_t a,size_t b) { return a < b ? a : b; }
void arion_menu_dispose(ArionPinyinMenu *menu) {
    arion_pinyin_candidates_dispose(menu->texts,menu->count);
    free(menu->indices);
    memset(menu,0,sizeof(*menu));
}
int arion_menu_load(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session,size_t index) {
    if (!menu->page_size || index == SIZE_MAX) return 0;
    while (index >= menu->count && !menu->exhausted) {
        size_t wanted = index + 1 - menu->count;
        size_t limit = wanted > menu->page_size * 3 ? wanted : menu->page_size * 3;
        char **more = NULL; size_t count = 0;
        if (!arion_pinyin_candidates(api,session,menu->fetched,limit,&more,&count)) return 0;
        if (count > SIZE_MAX / sizeof(char *) - menu->count || count > SIZE_MAX / sizeof(size_t) - menu->count) {
            arion_pinyin_candidates_dispose(more,count); return 0;
        }
        size_t total = menu->count + count;
        char **texts = (char **)calloc(total ? total : 1,sizeof(char *));
        size_t *indices = (size_t *)calloc(total ? total : 1,sizeof(size_t));
        if (!texts || !indices) { free(texts); free(indices); arion_pinyin_candidates_dispose(more,count); return 0; }
        if (menu->count) {
            memcpy(texts,menu->texts,menu->count*sizeof(char *));
            memcpy(indices,menu->indices,menu->count*sizeof(size_t));
        }
        for (size_t i=0;i<count;i++) { texts[menu->count+i]=more[i]; indices[menu->count+i]=menu->fetched+i; }
        free(more); free(menu->texts); free(menu->indices);
        menu->texts=texts; menu->indices=indices; menu->count=total;
        menu->fetched+=count; menu->exhausted=count<limit;
    }
    return 1;
}
int arion_menu_build(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session,size_t page_size,ArionMenuRank rank,void *context) {
    int expanded=menu->expanded;
    arion_menu_dispose(menu);
    if (page_size<1 || page_size>9) return 0;
    menu->page_size=page_size;
    menu->expanded=expanded;
    if (!arion_menu_load(menu,api,session,rank_window-1)) return 0;
    if (rank && menu->count) {
        size_t *order=(size_t *)calloc(menu->count,sizeof(size_t));
        char **texts=(char **)calloc(menu->count,sizeof(char *));
        unsigned char *seen=(unsigned char *)calloc(menu->count,1);
        int valid=order && texts && seen;
        if (valid) valid=rank(context,(const char *const *)menu->texts,menu->count,order);
        for (size_t i=0;valid && i<menu->count;i++) {
            if (order[i]>=menu->count || seen[order[i]]) { valid=0; break; }
            seen[order[i]]=1; texts[i]=menu->texts[order[i]];
        }
        free(seen);
        if (!valid) { free(order); free(texts); return 0; }
        free(menu->indices); free(menu->texts); menu->indices=order; menu->texts=texts;
    }
    return 1;
}
size_t arion_menu_visible_count(const ArionPinyinMenu *menu) {
    size_t first=menu->page*menu->page_size;
    return first<menu->count ? minimum(menu->page_size,menu->count-first) : 0;
}
size_t arion_menu_row_count(const ArionPinyinMenu *menu) {
    if (!menu->page_size || !menu->count) return 0;
    size_t pages=(menu->count-1)/menu->page_size+1;
    return menu->top_row<pages ? minimum(expanded_rows,pages-menu->top_row) : 0;
}
int arion_menu_prepare(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session) {
    if (menu->highlight<menu->count) arion_pinyin_highlight(api,session,menu->indices[menu->highlight],0);
    if (!arion_menu_load(menu,api,session,menu->page*menu->page_size+arion_menu_visible_count(menu))) return 0;
    return !menu->expanded || arion_menu_load(menu,api,session,(menu->top_row+expanded_rows)*menu->page_size);
}
int arion_menu_turn(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session,int delta) {
    if ((delta!=1 && delta!=-1) || (delta<0 && menu->page==0)) return 0;
    size_t page=delta>0 ? menu->page+1 : menu->page-1;
    if (page>SIZE_MAX/9-1 || !arion_menu_load(menu,api,session,page*menu->page_size) || page*menu->page_size>=menu->count) return 0;
    menu->page=page; menu->highlight=page*menu->page_size;
    if (page<menu->top_row) menu->top_row=page;
    if (page>=menu->top_row+expanded_rows) menu->top_row=page+1-expanded_rows;
    return 1;
}
int arion_menu_move(ArionPinyinMenu *menu,int delta) {
    if ((delta!=1 && delta!=-1) || (delta<0 && !menu->highlight)) return 0;
    size_t next=delta>0 ? menu->highlight+1 : menu->highlight-1;
    size_t first=menu->page*menu->page_size;
    if (next<first || next>=minimum(first+menu->page_size,menu->count)) return 0;
    menu->highlight=next; return 1;
}
int arion_menu_grid(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session,int expanded) {
    if (expanded && (!arion_menu_load(menu,api,session,(menu->page+1)*menu->page_size) || (menu->page+1)*menu->page_size>=menu->count)) return 0;
    menu->expanded=expanded!=0; menu->top_row=menu->page; return 1;
}
int arion_menu_engine_index(const ArionPinyinMenu *menu,size_t index,size_t row,int grid,size_t *engine_index) {
    if (!engine_index || index>=menu->page_size || (grid && (!menu->expanded || row>=arion_menu_row_count(menu)))) return 0;
    size_t at=(grid ? menu->top_row+row : menu->page)*menu->page_size+index;
    if (at>=menu->count) return 0;
    *engine_index=menu->indices[at]; return 1;
}
