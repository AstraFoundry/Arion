#pragma once
#include <stdint.h>
#include <stddef.h>
#ifdef __cplusplus
extern "C" {
#endif
// Replaces only the current unselected whole-input menu. Selection continues
// through Rime; this operation never commits text or persists a model proposal.
// order must permute the loaded engine prefix; its ranked order is retained.
int arion_completion_insert(uintptr_t session, const char *input, const char *text,
                            const size_t *order, size_t count);
#ifdef __cplusplus
}
#endif
