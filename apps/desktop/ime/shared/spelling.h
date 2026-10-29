#ifndef ARION_SHARED_SPELLING_H
#define ARION_SHARED_SPELLING_H

#ifdef __cplusplus
#include <cstdint>
#include <string>
namespace rime {
class Prism;
namespace corrector { class Corrections; }
}
namespace arion::ime {
// Bits follow the same stable order as the per-session Rime option names.
enum FuzzyPair : uint32_t {
    Z_ZH = 1u << 0, C_CH = 1u << 1, S_SH = 1u << 2,
    N_L = 1u << 3, F_H = 1u << 4, R_L = 1u << 5,
    AN_ANG = 1u << 6, EN_ENG = 1u << 7, IN_ING = 1u << 8,
    IAN_IANG = 1u << 9, UAN_UANG = 1u << 10
};
// True only for complete paths of normal, non-correction prism spellings.
bool normalSpelling(const rime::Prism& prism, const std::string& input);
void spellingCorrections(const rime::Prism& prism, const std::string& input,
                         bool typos, uint32_t fuzzy,
                         rime::corrector::Corrections* results);
}
extern "C" {
#endif
// Register after Rime initialization and before creating any session.
void arion_spelling_register(void);
#ifdef __cplusplus
}
#endif
#endif
