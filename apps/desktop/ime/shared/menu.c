#include "menu.h"
#include <stdlib.h>
#include <string.h>

enum { rank_window = 27, expanded_rows = 6 };
static size_t minimum(size_t a,size_t b) { return a < b ? a : b; }
static int english_word(const char *text) {
    if (!text || !text[0] || !text[1]) return 0;
    for (const unsigned char *c=(const unsigned char *)text;*c;c++)
        if (!((*c>='a' && *c<='z') || (*c>='A' && *c<='Z'))) return 0;
    return 1;
}
static void append_candidate(const ArionPinyinMenu *menu,char **texts,size_t *indices,
                             unsigned char *types,size_t *count,size_t position) {
    size_t output=(*count)++;
    texts[output]=menu->texts[position];
    indices[output]=menu->indices[position];
    types[output]=menu->emoji[position];
}
void arion_menu_dispose(ArionPinyinMenu *menu) {
    arion_pinyin_candidates_dispose(menu->texts,menu->count);
    free(menu->indices);
    free(menu->emoji);
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
        unsigned char *emoji = (unsigned char *)calloc(total ? total : 1,1);
        if (!texts || !indices || !emoji) { free(texts); free(indices); free(emoji); arion_pinyin_candidates_dispose(more,count); return 0; }
        if (menu->count) {
            memcpy(texts,menu->texts,menu->count*sizeof(char *));
            memcpy(indices,menu->indices,menu->count*sizeof(size_t));
            memcpy(emoji,menu->emoji,menu->count);
        }
        int valid=1;
        for (size_t i=0;i<count;i++) {
            texts[menu->count+i]=more[i]; indices[menu->count+i]=menu->fetched+i;
            if (menu->kind && !menu->kind(menu->kind_context,session,menu->fetched+i,&emoji[menu->count+i])) valid=0;
        }
        if (!valid) { free(texts); free(indices); free(emoji); arion_pinyin_candidates_dispose(more,count); return 0; }
        free(more); free(menu->texts); free(menu->indices); free(menu->emoji);
        menu->texts=texts; menu->indices=indices; menu->emoji=emoji; menu->count=total;
        menu->fetched+=count; menu->exhausted=count<limit;
    }
    return 1;
}
int arion_menu_build_typed(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session,size_t page_size,ArionMenuRank rank,void *context,
                          int (*kind)(void *,uintptr_t,size_t,unsigned char *),void *kind_context) {
    int expanded=menu->expanded;
    arion_menu_dispose(menu);
    if (page_size<1 || page_size>9) return 0;
    menu->page_size=page_size;
    menu->expanded=expanded;
    menu->kind=kind; menu->kind_context=kind_context;
    if (!arion_menu_load(menu,api,session,rank_window-1)) return 0;
    while (!menu->exhausted && menu->fetched < rank_window*3) {
        size_t text_count=0;
        for (size_t i=0;i<menu->count;i++) text_count+=!menu->emoji[i];
        if (text_count>=rank_window) break;
        if (!arion_menu_load(menu,api,session,menu->count+page_size*3-1)) return 0;
    }
    if (menu->count) {
        const size_t count=menu->count;
        size_t text_count=0, emoji_count=0;
        for (size_t i=0;i<count;i++) if (menu->emoji[i]) emoji_count++; else text_count++;
        char **ranked=(char **)calloc(text_count?text_count:1,sizeof(char *));
        size_t *text_positions=(size_t *)calloc(text_count?text_count:1,sizeof(size_t));
        size_t *emoji_positions=(size_t *)calloc(emoji_count?emoji_count:1,sizeof(size_t));
        size_t *order=(size_t *)calloc(text_count?text_count:1,sizeof(size_t));
        unsigned char *seen=(unsigned char *)calloc(text_count?text_count:1,1);
        char **texts=(char **)calloc(count,sizeof(char *));
        size_t *indices=(size_t *)calloc(count,sizeof(size_t));
        unsigned char *types=(unsigned char *)calloc(count,1);
        int valid=ranked && text_positions && emoji_positions && order && seen && texts && indices && types;
        size_t ti=0, ei=0;
        for (size_t i=0;valid && i<count;i++) {
            if (menu->emoji[i]) emoji_positions[ei++]=i;
            else { ranked[ti]=menu->texts[i]; text_positions[ti++]=i; }
        }
        if (valid && rank && text_count) valid=rank(context,(const char *const *)ranked,text_count,order);
        if (valid && !rank) for (size_t i=0;i<text_count;i++) order[i]=i;
        for (size_t i=0;valid && i<text_count;i++) {
            if (order[i]>=text_count || seen[order[i]]) valid=0;
            else seen[order[i]]=1;
        }
        size_t out=0;
        if (valid) {
            size_t protected=minimum(3,text_count);
            // English preview candidates keep their first-page slots ahead of Emoji.
            for (size_t i=protected;i<minimum(5,text_count);i++)
                if (english_word(ranked[order[i]])) protected=i+1;
            for (size_t i=0;i<protected;i++) append_candidate(menu,texts,indices,types,&out,text_positions[order[i]]);
            size_t emoji_slots=page_size>protected?page_size-protected:(protected>3?2:0);
            size_t first_emoji=minimum(emoji_count,minimum(2,emoji_slots));
            for (size_t i=0;i<first_emoji;i++) append_candidate(menu,texts,indices,types,&out,emoji_positions[i]);
            for (size_t i=protected;i<text_count;i++) append_candidate(menu,texts,indices,types,&out,text_positions[order[i]]);
            for (size_t i=first_emoji;i<emoji_count;i++) append_candidate(menu,texts,indices,types,&out,emoji_positions[i]);
        }
        free(ranked); free(text_positions); free(emoji_positions); free(order); free(seen);
        if (!valid || out!=count) { free(texts); free(indices); free(types); return 0; }
        free(menu->texts); free(menu->indices); free(menu->emoji);
        menu->texts=texts; menu->indices=indices; menu->emoji=types;
    }
    return 1;
}
int arion_menu_build(ArionPinyinMenu *menu,RimeApi *api,uintptr_t session,size_t page_size,ArionMenuRank rank,void *context) {
    return arion_menu_build_typed(menu,api,session,page_size,rank,context,NULL,NULL);
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
