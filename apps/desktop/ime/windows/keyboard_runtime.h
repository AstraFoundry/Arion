#pragma once
#include <stddef.h>
#include <stdint.h>
#include <rime_api.h>
#include "../shared/emoji.h"
#include "../shared/personal_vocabulary_bridge.h"

#ifdef ARION_KEYBOARD_BUILD
#define ARION_KEYBOARD_EXPORT __declspec(dllexport)
#else
#define ARION_KEYBOARD_EXPORT
#endif
#ifdef __cplusplus
extern "C" {
#endif
// The host verifies the immutable payload before loading this module.
// setup -> encrypted provider -> initialize; never deploy or plaintext userdb.
ARION_KEYBOARD_EXPORT RimeApi *arion_keyboard_start(const char *data,const char *user,char *error,size_t capacity);
ARION_KEYBOARD_EXPORT void arion_keyboard_enter(void);
ARION_KEYBOARD_EXPORT void arion_keyboard_leave(void);
ARION_KEYBOARD_EXPORT int arion_keyboard_error(char *error,size_t capacity);
ARION_KEYBOARD_EXPORT int arion_keyboard_reset_learning(char *error,size_t capacity);
ARION_KEYBOARD_EXPORT uint64_t arion_keyboard_generation(void);
ARION_KEYBOARD_EXPORT int arion_keyboard_order(const char *const *candidates,size_t count,const unsigned char *eligible,const char *previous,size_t *indices,char *error,size_t capacity);
ARION_KEYBOARD_EXPORT int arion_keyboard_learn(const char *commit,const char *previous,char *error,size_t capacity);
ARION_KEYBOARD_EXPORT int arion_keyboard_contains_han(const char *text,size_t length);
ARION_KEYBOARD_EXPORT int arion_keyboard_context(const char *commit,char *context,size_t capacity);
ARION_KEYBOARD_EXPORT int arion_keyboard_candidate_metadata(uintptr_t session,size_t index,ArionCandidateMetadata *metadata);
ARION_KEYBOARD_EXPORT int arion_keyboard_refresh_emoji(uintptr_t session);
ARION_KEYBOARD_EXPORT int arion_keyboard_restore_personal(uintptr_t session, uint64_t *revision);
ARION_KEYBOARD_EXPORT int arion_keyboard_publish_personal(uintptr_t session, uint64_t revision, const ArionPersonalTerm *terms, size_t count);
ARION_KEYBOARD_EXPORT void arion_keyboard_stop(void);
#ifdef __cplusplus
}
#endif
