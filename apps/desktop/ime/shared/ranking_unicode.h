#pragma once
#include <cstddef>
#include <string>
#include <vector>
namespace arion::ime {
// Boundaries include zero and the byte length; malformed UTF-8 is rejected.
bool rankingGraphemes(const std::string &, std::vector<size_t> &);
// Returns 1 for Script=Han, 0 for no Han, and -1 for malformed UTF-8.
// The entire string is validated, including bytes following a Han character.
int rankingContainsHan(const std::string &);
std::string rankingTrimmed(const std::string &);
}
