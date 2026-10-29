#include "pinyin.h"
#include <limits.h>
#include <stdlib.h>
#include <string.h>

int arion_pinyin_utf16_position(const char *text, size_t bytes, size_t *position) {
    size_t i = 0, units = 0;
    if (!text || !position || bytes > strlen(text)) return 0;
    while (i < bytes) {
        unsigned char first = (unsigned char)text[i++];
        unsigned int value, minimum;
        size_t remaining;
        if (first < 0x80) { units++; continue; }
        if (first >= 0xc2 && first <= 0xdf) { value = first & 31; remaining = 1; minimum = 0x80; }
        else if (first >= 0xe0 && first <= 0xef) { value = first & 15; remaining = 2; minimum = 0x800; }
        else if (first >= 0xf0 && first <= 0xf4) { value = first & 7; remaining = 3; minimum = 0x10000; }
        else return 0;
        if (remaining > bytes - i) return 0;
        while (remaining--) {
            unsigned char next = (unsigned char)text[i++];
            if ((next & 0xc0) != 0x80) return 0;
            value = (value << 6) | (next & 63);
        }
        if (value < minimum || value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff)) return 0;
        units += value > 0xffff ? 2 : 1;
    }
    *position = units;
    return 1;
}

static char *copy_text(const char *text) {
    size_t units;
    if (!text) text = "";
    size_t length = strlen(text);
    if (!arion_pinyin_utf16_position(text, length, &units)) return NULL;
    char *copy = (char *)malloc(length + 1);
    if (copy) memcpy(copy, text, length + 1);
    return copy;
}

int arion_pinyin_api_supported(RimeApi *api) {
    return api && RIME_PROVIDED(api, highlight_candidate_on_current_page) &&
        RIME_PROVIDED(api, change_page) && RIME_PROVIDED(api, highlight_candidate) &&
        RIME_PROVIDED(api, candidate_list_from_index);
}
uintptr_t arion_pinyin_create_session(RimeApi *api) {
    if (!arion_pinyin_api_supported(api)) return 0;
    RimeSessionId session = api->create_session();
    if (session && !api->select_schema(session, "arion_pinyin")) {
        api->destroy_session(session);
        return 0;
    }
    return session;
}
void arion_pinyin_destroy_session(RimeApi *api, uintptr_t session) {
    if (session) api->destroy_session(session);
}
void arion_pinyin_set_options(RimeApi *api, uintptr_t session, int ascii, int full_width, int english_punctuation) {
    if (!session) return;
    api->set_option(session, "ascii_mode", ascii != 0);
    api->set_option(session, "full_shape", full_width && !ascii);
    api->set_option(session, "ascii_punct", english_punctuation != 0);
}
int arion_pinyin_process_key(RimeApi *api, uintptr_t session, int key, int modifiers) {
    if (!session || key < 0) return 0;
    int handled = api->process_key(session, key, modifiers);
    /* Rime Escape can remove only the unfinished suffix after a selection.
     * Arion cancellation discards the entire still-provisional composition. */
    if (handled && key == 0xff1b && modifiers == 0) api->clear_composition(session);
    return handled;
}
int arion_pinyin_commit_raw(RimeApi *api, uintptr_t session) {
    if (!session) return 0;
    int full_width = api->get_option(session, "full_shape");
    if (full_width) api->set_option(session, "full_shape", 0);
    int handled = arion_pinyin_process_key(api, session, 0xff0d, 0); /* Return */
    if (full_width) api->set_option(session, "full_shape", 1);
    return handled;
}
int arion_pinyin_select(RimeApi *api, uintptr_t session, size_t index, int current_page) {
    return session && (current_page ? api->select_candidate_on_current_page(session, index) : api->select_candidate(session, index));
}
int arion_pinyin_highlight(RimeApi *api, uintptr_t session, size_t index, int current_page) {
    return session && (current_page ? api->highlight_candidate_on_current_page(session, index) : api->highlight_candidate(session, index));
}
void arion_pinyin_clear(RimeApi *api, uintptr_t session) {
    if (session) api->clear_composition(session);
}
void arion_pinyin_candidates_dispose(char **texts, size_t count) {
    if (!texts) return;
    for (size_t i = 0; i < count; i++) free(texts[i]);
    free(texts);
}
int arion_pinyin_candidates(RimeApi *api, uintptr_t session, size_t start, size_t limit, char ***texts, size_t *count) {
    *texts = NULL; *count = 0;
    if (!session || start > INT_MAX || limit > SIZE_MAX / sizeof(char *)) return 0;
    if (!limit) return 1;
    RimeCandidateListIterator iterator = {0};
    if (!api->candidate_list_from_index(session, &iterator, (int)start)) return 1;
    char **result = (char **)calloc(limit, sizeof(char *));
    int valid = result != NULL;
    while (valid && *count < limit && api->candidate_list_next(&iterator)) {
        result[*count] = copy_text(iterator.candidate.text);
        valid = result[*count] != NULL;
        if (valid) (*count)++;
    }
    api->candidate_list_end(&iterator);
    if (!valid) { arion_pinyin_candidates_dispose(result, *count); *count = 0; return 0; }
    *texts = result;
    return 1;
}
void arion_pinyin_snapshot_dispose(ArionPinyinSnapshot *snapshot) {
    free(snapshot->preedit); free(snapshot->commit);
    arion_pinyin_candidates_dispose(snapshot->candidates, snapshot->count);
    arion_pinyin_candidates_dispose(snapshot->comments, snapshot->count);
    memset(snapshot, 0, sizeof(*snapshot));
}
int arion_pinyin_snapshot(RimeApi *api, uintptr_t session, ArionPinyinSnapshot *out) {
    memset(out, 0, sizeof(*out));
    if (!session) return 4;
    RIME_STRUCT(RimeContext, context);
    RIME_STRUCT(RimeStatus, status);
    if (!api->get_context(session, &context)) return 5;
    if (!api->get_status(session, &status)) { api->free_context(&context); return 5; }
    out->preedit = copy_text(context.composition.preedit);
    out->ascii = status.is_ascii_mode;
    int valid = out->preedit != NULL && context.menu.num_candidates >= 0 &&
        context.menu.num_candidates <= 9 && context.menu.page_no >= 0;
    if (valid && out->preedit[0]) valid = context.composition.cursor_pos >= 0 &&
        arion_pinyin_utf16_position(out->preedit, (size_t)context.composition.cursor_pos, &out->cursor_utf16);
    if (valid) {
        out->count = (size_t)context.menu.num_candidates;
        out->candidates = (char **)calloc(out->count ? out->count : 1, sizeof(char *));
        out->comments = (char **)calloc(out->count ? out->count : 1, sizeof(char *));
        valid = out->candidates && out->comments && (!out->count || context.menu.candidates);
        for (size_t i = 0; valid && i < out->count; i++) {
            out->candidates[i] = copy_text(context.menu.candidates[i].text);
            out->comments[i] = copy_text(context.menu.candidates[i].comment);
            valid = out->candidates[i] && out->comments[i];
        }
    }
    out->page = context.menu.page_no > 0 ? (size_t)context.menu.page_no : 0;
    out->page_size = context.menu.page_size > 0 ? (size_t)context.menu.page_size : 0;
    out->last_page = context.menu.is_last_page;
    out->highlighted = context.menu.highlighted_candidate_index > 0 ? (size_t)context.menu.highlighted_candidate_index : 0;
    api->free_context(&context); api->free_status(&status);
    if (!valid) { arion_pinyin_snapshot_dispose(out); return 6; }
    RIME_STRUCT(RimeCommit, commit);
    if (api->get_commit(session, &commit)) {
        out->commit = copy_text(commit.text);
        api->free_commit(&commit);
    } else out->commit = copy_text("");
    if (!out->commit) { arion_pinyin_snapshot_dispose(out); return 6; }
    return 0;
}
