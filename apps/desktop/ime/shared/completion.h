#pragma once
#include <stdint.h>
#ifdef __cplusplus
extern "C" {
#endif
// Replaces only the current unselected whole-input menu. Selection continues
// through Rime; this operation never commits text or persists a model proposal.
int arion_completion_insert(uintptr_t session, const char *input, const char *text);
#ifdef __cplusplus
}
#endif
