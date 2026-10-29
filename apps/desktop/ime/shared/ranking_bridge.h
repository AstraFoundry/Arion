#pragma once
#include <stddef.h>
#ifdef __cplusplus
extern "C" {
#endif
typedef struct arion_ranker arion_ranker;
typedef void (*arion_ranker_change)(const char *key,const char *value,void *context);
arion_ranker *arion_ranker_create(const char *const *keys,const char *const *values,size_t count);
void arion_ranker_destroy(arion_ranker *);
size_t arion_ranker_count(const arion_ranker *);
// Returned UTF-8 context is allocated with malloc; caller frees it.
char *arion_ranker_context(const char *commit);
// Explicit byte length preserves embedded NULs. Returns 1 for Han, 0 for no
// Han, and -1 for a null pointer or malformed UTF-8; callers must compare ==1.
int arion_ranker_contains_han(const char *text,size_t length);
void arion_ranker_order(const arion_ranker *,const char *const *candidates,size_t count,const unsigned char *eligible,const char *previous,size_t *indices);
void arion_ranker_learn(arion_ranker *,const char *commit,const char *previous,arion_ranker_change,void *context);
#ifdef __cplusplus
}
#endif
