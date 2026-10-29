#ifndef ARION_SHARED_SPELLING_OPTIONS_H
#define ARION_SHARED_SPELLING_OPTIONS_H
#include <rime_api.h>
#include <stdint.h>

enum { ARION_FUZZY_COUNT = 11, ARION_FUZZY_MASK = (1u << ARION_FUZZY_COUNT) - 1 };
static const char *const arion_fuzzy_options[ARION_FUZZY_COUNT] = {
    "arion_fuzzy_z_zh", "arion_fuzzy_c_ch", "arion_fuzzy_s_sh",
    "arion_fuzzy_n_l", "arion_fuzzy_f_h", "arion_fuzzy_r_l",
    "arion_fuzzy_an_ang", "arion_fuzzy_en_eng", "arion_fuzzy_in_ing",
    "arion_fuzzy_ian_iang", "arion_fuzzy_uan_uang"
};

// Apply through the public Rime ABI so both hosts use one option mapping.
// Avoid re-composing when an option already has the requested value.
static inline void arion_spelling_apply(RimeApi *api, uintptr_t session, int typo, uint32_t fuzzy) {
    if (!api || !session) return;
    if (!!api->get_option(session,"arion_typo") != !!typo)
        api->set_option(session,"arion_typo",!!typo);
    for (unsigned i=0; i<ARION_FUZZY_COUNT; ++i) {
        int enabled=(fuzzy & (1u<<i))!=0;
        if (!!api->get_option(session,arion_fuzzy_options[i]) != enabled)
            api->set_option(session,arion_fuzzy_options[i],enabled);
    }
}
#endif
