#pragma once
#include <stddef.h>
#include <stdint.h>
#ifdef __cplusplus
extern "C" {
#endif
typedef struct {
    const char *id;
    const char *text;
    const char *reading;
} ArionPersonalTerm;
// Call on the session's serialized engine thread, only between compositions.
// This in-memory projection does not persist terms or mutate learned phrases.
// Returns 1 on acceptance, 0 on invalid/stale data, missing session or busy input.
int arion_personal_vocabulary_replace(uintptr_t session, uint64_t revision,
                                      const ArionPersonalTerm *terms, size_t count);
// Restore the encrypted snapshot into an idle session. Missing cache is empty.
int arion_personal_vocabulary_restore(uintptr_t session, uint64_t *revision);
// Persist before exposing a newer projection. Failure leaves active terms intact.
int arion_personal_vocabulary_publish(uintptr_t session, uint64_t revision,
                                      const ArionPersonalTerm *terms, size_t count);
#ifdef __cplusplus
}
#endif
