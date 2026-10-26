#pragma once
#include "learning.h"
namespace rime { class Db; }
namespace arion::ime {
// Called once before Rime initialization; the registry owns its component.
bool registerEncryptedLearning(std::shared_ptr<LearningProvider>, LearningError &);
LearningError encryptedLearningError();
void clearEncryptedLearningError();
rime::Db *makeEncryptedRimeDatabase(std::shared_ptr<LearningProvider>, const std::string &file, const std::string &name);
std::shared_ptr<LearningProvider> registeredLearningProvider();
std::unique_ptr<LearningDatabase> openContextLearning(LearningError &);
}
extern "C" {
int arion_learning_windows_register(const char *directory, char *error, size_t errorCapacity);
int arion_learning_windows_reset(const char *directory, char *error, size_t errorCapacity);
int arion_learning_error(char *error, size_t errorCapacity);
}
