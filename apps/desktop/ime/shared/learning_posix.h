#pragma once
#include "learning.h"
#include <memory>
#include <string>

namespace arion::ime {
// Authenticated encryption supplied by the host OS (CryptoKit, Android
// Keystore). The label binds a snapshot to its database name.
class LearningCipher {
public:
    virtual ~LearningCipher() = default;
    virtual bool seal(const std::string &label, const LearningBytes &plain, LearningBytes &sealed) = 0;
    virtual bool open(const std::string &label, const LearningBytes &sealed, LearningBytes &plain) = 0;
};

// A learning vault on a POSIX file system (macOS, iOS, Android): advisory
// flock leases, fsync'd atomic replacement and the ARIONUD1 envelope around
// the cipher's output. A null cipher can only reset or validate the vault.
std::shared_ptr<LearningProvider> makePosixLearningProvider(const std::string &directory,
    std::shared_ptr<LearningCipher> cipher, const std::string &labelPrefix);

// Whether a vault entry may hold learning data: anything but lock files and
// Finder metadata. A vault with data never authorizes creating a new key.
bool learningVaultEntryIsData(const std::string &name);
} // namespace arion::ime
