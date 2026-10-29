#include "pinyin.h"
#include "input.h"
#include "menu.h"
#include <assert.h>
#include <stdio.h>
#include <string.h>

static char text[] = "\xe4\xbd\xa0\xf0\x9f\x98\x80";
static RimeCandidate candidate;
static int invalid_cursor, invalid_candidate, commit_pending, contexts_freed, statuses_freed, commits_freed;
static int candidate_count = 40;
static const char *const *menu_fixture;
static Bool context_read(RimeSessionId session, RimeContext *context) {
    assert(session == 42);
    context->composition.preedit = text;
    context->composition.cursor_pos = invalid_cursor ? 4 : 7;
    context->menu.num_candidates = 1;
    context->menu.page_no = 2;
    context->menu.page_size = 9;
    context->menu.candidates = &candidate;
    candidate.text = invalid_candidate ? "\xc0\x80" : "\xe4\xbd\xa0";
    candidate.comment = "";
    context->menu.is_last_page = 1;
    return True;
}
static Bool status_read(RimeSessionId session, RimeStatus *status) {
    assert(session == 42); status->is_ascii_mode = False; return True;
}
static Bool context_free(RimeContext *context) { (void)context; contexts_freed++; return True; }
static Bool status_free(RimeStatus *status) { (void)status; statuses_freed++; return True; }
static Bool commit_read(RimeSessionId session, RimeCommit *commit) {
    assert(session == 42);
    if (!commit_pending) return False;
    commit_pending = 0; commit->text = text; return True;
}
static Bool commit_free(RimeCommit *commit) { (void)commit; commits_freed++; return True; }
static Bool list_from(RimeSessionId session,RimeCandidateListIterator *iterator,int index) {
    (void)session; iterator->index=index-1; return True;
}
static Bool list_next(RimeCandidateListIterator *iterator) {
    if (++iterator->index>=candidate_count) return False;
    iterator->candidate.text=menu_fixture ? (char *)menu_fixture[iterator->index] : text;
    iterator->candidate.comment=""; return True;
}
static void list_end(RimeCandidateListIterator *iterator) { (void)iterator; }
static Bool highlight_candidate(RimeSessionId session, size_t index) {
    assert(session==42 && index<(size_t)candidate_count); return True;
}
static int reverse_order(void *context,const char *const *texts,size_t count,const unsigned char *eligible,size_t *order) {
    (void)context; (void)texts; (void)eligible;
    for (size_t i=0;i<count;i++) order[i]=count-i-1;
    return 1;
}
static int invalid_order(void *context,const char *const *texts,size_t count,const unsigned char *eligible,size_t *order) {
    (void)context; (void)texts; (void)eligible;
    for (size_t i=0;i<count;i++) order[i]=0;
    return 1;
}
static int fixture_kind(void *context,uintptr_t session,size_t index,ArionCandidateMetadata *metadata) {
    (void)context; (void)session; memset(metadata,0,sizeof(*metadata)); metadata->end=7; metadata->emoji=index==1; return 1;
}

static int candidate_metadata(void *context,uintptr_t session,size_t index,ArionCandidateMetadata *out) {
    (void)context; (void)session; (void)index;
    memset(out,0,sizeof(*out)); out->end=4; return 1;
}

static int mixed_metadata(void *context,uintptr_t session,size_t index,ArionCandidateMetadata *out) {
    candidate_metadata(context,session,index,out);
    if (index==1) out->emoji=1;
    if (index==2) out->end=2;
    if (index==3) out->start=2;
    return 1;
}
static int check_ranges(void *context,const char *const *texts,size_t count,const unsigned char *eligible,size_t *order) {
    (void)context; (void)texts;
    assert(count==4 && eligible[0] && !eligible[1] && !eligible[2] && eligible[3]);
    order[0]=3; order[1]=0; order[2]=1; order[3]=2;
    return 1;
}

int main(void) {
    char *rank_texts[]={"first","english","second","emoji","third"};
    size_t rank_indices[]={10,11,12,13,14};
    ArionCandidateMetadata rank_metadata[5]={0};
    for (size_t i=0;i<5;++i) rank_metadata[i].end=4;
    rank_metadata[3].emoji=1;
    ArionPinyinMenu ranked={0};
    ranked.texts=rank_texts; ranked.indices=rank_indices; ranked.metadata=rank_metadata; ranked.count=5;
    size_t slots[]={0,2,4};
    assert(arion_menu_promote_slots(&ranked,slots,3,2));
    assert(rank_indices[0]==14 && rank_indices[1]==11 && rank_indices[2]==10 && rank_indices[3]==13 && rank_indices[4]==12);
    ranked.highlight=1;
    assert(!arion_menu_promote_slots(&ranked,slots,3,1));
    ranked.highlight=0; rank_metadata[2].end=2;
    assert(!arion_menu_promote_slots(&ranked,slots,3,1));
    assert(rank_indices[0]==14);
    size_t position = 99;
    assert(arion_pinyin_utf16_position(text, 7, &position) && position == 3);
    assert(arion_pinyin_utf16_position(text, 3, &position) && position == 1);
    assert(!arion_pinyin_utf16_position(text, 4, &position));
    assert(!arion_pinyin_utf16_position("\xed\xa0\x80", 3, &position));
    assert(!arion_pinyin_utf16_position("\xf4\x90\x80\x80", 4, &position));
    assert(!arion_pinyin_utf16_position("\xc0\x80", 2, &position));

    RIME_STRUCT(RimeApi, api);
    api.get_context = context_read; api.free_context = context_free;
    api.get_status = status_read; api.free_status = status_free;
    api.get_commit = commit_read; api.free_commit = commit_free;
    ArionPinyinSnapshot snapshot;
    commit_pending = 1;
    assert(arion_pinyin_snapshot(&api, 42, &snapshot) == 0);
    assert(snapshot.cursor_utf16 == 3 && snapshot.count == 1);
    assert(snapshot.page == 2 && snapshot.page_size == 9 && snapshot.last_page);
    assert(strcmp(snapshot.commit, text) == 0 && snapshot.commit != text);
    assert(snapshot.candidates[0] != candidate.text);
    arion_pinyin_snapshot_dispose(&snapshot);
    assert(arion_pinyin_snapshot(&api, 42, &snapshot) == 0);
    assert(snapshot.commit[0] == 0 && commits_freed == 1);
    arion_pinyin_snapshot_dispose(&snapshot);
    invalid_cursor = 1;
    assert(arion_pinyin_snapshot(&api, 42, &snapshot) == 6);
    invalid_cursor = 0; invalid_candidate = 1;
    assert(arion_pinyin_snapshot(&api, 42, &snapshot) == 6);
    assert(contexts_freed == 4 && statuses_freed == 4);
    assert(snapshot.preedit == NULL && snapshot.count == 0);

    ArionShiftTap tap = {0};
    assert(!arion_shift_transition(&tap, 1, 1, 1, 0, 0, 1.0));
    assert(!arion_shift_transition(&tap, 1, 1, 1, 0, 1, 1.05));
    assert(arion_shift_transition(&tap, 1, 0, 1, 0, 0, 1.1));
    assert(!arion_shift_transition(&tap, 1, 1, 1, 0, 0, 2.0));
    arion_shift_interrupt(&tap);
    assert(!arion_shift_transition(&tap, 1, 0, 1, 0, 0, 2.1));
    assert(!arion_shift_transition(&tap, 1, 1, 1, 0, 0, 3.0));
    assert(!arion_shift_transition(&tap, 1, 1, 2, 0, 1, 3.05));
    assert(!arion_shift_transition(&tap, 1, 0, 2, 0, 0, 3.1));
    assert(!arion_shift_transition(&tap, 1, 1, 1, 0, 0, 4.0));
    assert(!arion_shift_transition(&tap, 1, 0, 1, 0, 0, 4.31));
    assert(!arion_shift_transition(&tap, 1, 1, 1, 0, 0, 5.0));
    arion_shift_reset(&tap);
    assert(!arion_shift_transition(&tap, 1, 0, 1, 0, 0, 5.1));
    assert(arion_key_action(' ', 1, 1) == ARION_KEY_TOGGLE_ASCII);
    assert(arion_key_action('.', 4, 0) == ARION_KEY_TOGGLE_PUNCTUATION);
    assert(arion_key_action('.', 4, 1) == ARION_KEY_PASS);
    assert(arion_key_action('c', 4, 0) == ARION_KEY_PASS);
    assert(arion_key_action('a', 0, 0) == ARION_KEY_PROCESS);
    assert(arion_key_normalize(0x00fc, 0, 0) == 'v');
    assert(arion_key_normalize(0x00dc, 1, 0) == 'V');
    assert(arion_key_normalize(0x00fc, 0, 1) == 0x00fc);
    assert(arion_key_normalize(0x00dc, 1, 1) == 0x00dc);
    for (int modifier = 4; modifier <= 8; modifier *= 2)
        assert(arion_key_normalize(0x00fc, modifier, 0) == 0x00fc);
    assert(arion_key_normalize(0x00dc, 1 << 26, 0) == 0x00dc);
    assert(arion_key_normalize('v', 0, 0) == 'v');
    api.candidate_list_from_index=list_from; api.candidate_list_next=list_next; api.candidate_list_end=list_end;
    api.highlight_candidate=highlight_candidate;
    ArionPinyinMenu menu={0}; size_t engine_index=0;
    assert(arion_menu_build_typed(&menu,&api,42,5,reverse_order,NULL,candidate_metadata,NULL));
    assert(menu.count==27 && menu.indices[0]==26 && arion_menu_visible_count(&menu)==5);
    assert(arion_menu_engine_index(&menu,0,0,0,&engine_index) && engine_index==26);
    assert(arion_menu_turn(&menu,&api,42,1) && menu.page==1 && menu.highlight==5);
    assert(arion_menu_grid(&menu,&api,42,1) && menu.top_row==1);
    assert(arion_menu_load(&menu,&api,42,36) && menu.count==40 && menu.indices[27]==27);
    assert(arion_menu_row_count(&menu)==6);
    assert(arion_menu_engine_index(&menu,1,1,1,&engine_index) && engine_index==15);
    assert(!arion_menu_engine_index(&menu,5,0,0,&engine_index));
    assert(arion_menu_build_typed(&menu,&api,42,5,reverse_order,NULL,candidate_metadata,NULL));
    assert(menu.expanded && menu.page==0 && menu.top_row==0 && menu.highlight==0);
    assert(menu.indices[0]==26 && arion_menu_prepare(&menu,&api,42));
    candidate_count=3;
    assert(arion_menu_build_typed(&menu,&api,42,5,reverse_order,NULL,candidate_metadata,NULL));
    assert(menu.expanded && menu.count==3 && arion_menu_prepare(&menu,&api,42));
    assert(arion_menu_row_count(&menu)==1 && arion_menu_visible_count(&menu)==3);
    assert(arion_menu_engine_index(&menu,2,0,1,&engine_index) && engine_index==0);
    assert(arion_menu_grid(&menu,&api,42,0));
    assert(arion_menu_build(&menu,&api,42,5,NULL,NULL) && !menu.expanded);
    candidate_count=40;
    assert(arion_menu_build(&menu,&api,42,5,NULL,NULL));
    assert(arion_menu_grid(&menu,&api,42,1) && menu.expanded);
    arion_menu_dispose(&menu);
    assert(arion_menu_build(&menu,&api,42,5,NULL,NULL) && !menu.expanded);
    assert(!arion_menu_build_typed(&menu,&api,42,5,invalid_order,NULL,candidate_metadata,NULL));
    arion_menu_dispose(&menu);
    candidate_count=5;
    assert(arion_menu_build_typed(&menu,&api,42,5,check_ranges,NULL,mixed_metadata,NULL));
    assert(menu.indices[0]==4 && menu.metadata[0].start==0 && menu.metadata[0].end==4);
    assert(menu.indices[3]==1 && menu.metadata[3].emoji);
    arion_menu_dispose(&menu);
    assert(!arion_menu_build(&menu,&api,42,5,reverse_order,NULL));
    arion_menu_dispose(&menu);
    const char *const bilingual[]={"白领","🧑‍💼","百灵","白令","bailing","Bailing"};
    menu_fixture=bilingual; candidate_count=6;
    assert(arion_menu_build_typed(&menu,&api,42,5,NULL,NULL,fixture_kind,NULL));
    assert(strcmp(menu.texts[3],"bailing")==0 && strcmp(menu.texts[4],"Bailing")==0);
    assert(arion_menu_engine_index(&menu,3,0,0,&engine_index) && engine_index==4);
    assert(menu.metadata[5].emoji && strcmp(menu.texts[5],"🧑‍💼")==0);
    arion_menu_dispose(&menu);
    puts("Shared Pinyin ownership, UTF boundaries, commit consumption and key policy tests passed.");
    return 0;
}
