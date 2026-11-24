#include "learning_posix.h"
#include <cerrno>
#include <cstdlib>
#include <cstring>
#include <dirent.h>
#include <fcntl.h>
#include <sys/file.h>
#include <sys/stat.h>
#include <unistd.h>

namespace arion::ime {
namespace {
bool failed(LearningError &error, int code, const char *operation) { error = {code, operation}; return false; }

std::string parentOf(const std::string &file) {
    auto slash = file.find_last_of('/');
    return slash == std::string::npos ? std::string(".") : slash == 0 ? std::string("/") : file.substr(0, slash);
}

bool flushDirectory(const std::string &directory) {
    int fd = open(directory.c_str(), O_RDONLY | O_DIRECTORY);
    if (fd < 0) return false;
    bool ok = fsync(fd) == 0;
    if (close(fd)) ok = false;
    return ok;
}

class FileLocks final : public LearningLock {
public:
    std::vector<int> files;
    ~FileLocks() override { for (int fd : files) { flock(fd, LOCK_UN); close(fd); } }
};

class PosixProvider final : public LearningProvider {
    std::string directory_, labelPrefix_;
    std::shared_ptr<LearningCipher> cipher_;
    bool hold(const std::string &file, bool exclusive, FileLocks &held, LearningError &error) {
        int fd = open(file.c_str(), O_RDWR | O_CREAT | O_NOFOLLOW, 0600);
        struct stat info{};
        if (fd < 0) return failed(error, 8, "lock open");
        if (fstat(fd, &info) || !S_ISREG(info.st_mode) || flock(fd, (exclusive ? LOCK_EX : LOCK_SH) | LOCK_NB)) {
            close(fd);
            return failed(error, 8, "lock acquisition");
        }
        held.files.push_back(fd);
        return true;
    }
public:
    PosixProvider(std::string directory, std::shared_ptr<LearningCipher> cipher, std::string labelPrefix)
        : directory_(std::move(directory)), labelPrefix_(std::move(labelPrefix)), cipher_(std::move(cipher)) {}
    const std::string &directory() const override { return directory_; }
    bool lock(const std::string &file, bool readonly, bool reset, std::unique_ptr<LearningLock> &held, LearningError &error) override {
        auto locks = std::make_unique<FileLocks>();
        if (!hold(learningPath(directory_, ".learning.lock"), reset, *locks, error) ||
            (!file.empty() && !hold(file + ".lock", !readonly, *locks, error))) return false;
        held = std::move(locks);
        return true;
    }
    bool read(const std::string &file, LearningBytes &bytes, bool &missing, LearningError &error) override {
        missing = false;
        int fd = open(file.c_str(), O_RDONLY | O_NOFOLLOW);
        if (fd < 0) {
            if (errno == ENOENT) { missing = true; bytes.clear(); return true; }
            return failed(error, 5, "snapshot open");
        }
        struct stat info{};
        if (fstat(fd, &info) || !S_ISREG(info.st_mode) || info.st_size < 0 ||
            static_cast<uint64_t>(info.st_size) > maximumLearningSnapshot + 64) {
            close(fd);
            return failed(error, 5, "snapshot size");
        }
        LearningBytes next(static_cast<size_t>(info.st_size));
        size_t offset = 0;
        while (offset < next.size()) {
            ssize_t count = ::read(fd, next.data() + offset, next.size() - offset);
            if (count < 0 && errno == EINTR) continue;
            if (count <= 0) { close(fd); return failed(error, 5, "snapshot read"); }
            offset += static_cast<size_t>(count);
        }
        if (close(fd)) return failed(error, 5, "snapshot close");
        bytes = std::move(next);
        return true;
    }
    bool writeAtomic(const std::string &file, const LearningBytes &bytes, LearningError &error) override {
        const std::string directory = parentOf(file);
        std::string temporary = learningPath(directory, ".encrypted-XXXXXX");
        // mkstemp creates the file exclusively with mode 0600.
        int fd = mkstemp(temporary.data());
        if (fd < 0) return failed(error, 6, "temporary create");
        size_t offset = 0;
        bool ok = true;
        while (offset < bytes.size()) {
            ssize_t count = ::write(fd, bytes.data() + offset, bytes.size() - offset);
            if (count < 0 && errno == EINTR) continue;
            if (count <= 0) { ok = false; break; }
            offset += static_cast<size_t>(count);
        }
        if (ok && fsync(fd)) ok = false;
        if (close(fd)) ok = false;
        if (ok && rename(temporary.c_str(), file.c_str())) ok = false;
        if (!ok) { unlink(temporary.c_str()); return failed(error, 6, "snapshot replace"); }
        return flushDirectory(directory) || failed(error, 6, "vault flush");
    }
    bool remove(const std::string &file, LearningError &error) override {
        if (unlink(file.c_str()) && errno != ENOENT) return failed(error, 5, "snapshot remove");
        return flushDirectory(parentOf(file)) || failed(error, 5, "vault flush");
    }
    bool files(std::vector<std::string> &names, LearningError &error) override {
        DIR *directory = opendir(directory_.c_str());
        if (!directory) return failed(error, 5, "vault enumeration");
        names.clear();
        errno = 0;
        while (dirent *entry = readdir(directory)) {
            std::string name = entry->d_name;
            if (name != "." && name != "..") names.push_back(std::move(name));
        }
        bool ok = errno == 0;
        if (closedir(directory)) ok = false;
        return ok || failed(error, 5, "vault enumeration");
    }
    bool seal(const std::string &name, const LearningBytes &plain, LearningBytes &sealed, LearningError &error) override {
        if (!cipher_) return failed(error, 1, "key unavailable");
        LearningBytes encrypted;
        if (!cipher_->seal(labelPrefix_ + name, plain, encrypted)) return failed(error, 2, "AES-GCM authentication");
        sealed = {'A', 'R', 'I', 'O', 'N', 'U', 'D', '1'};
        sealed.insert(sealed.end(), encrypted.begin(), encrypted.end());
        return true;
    }
    bool unseal(const std::string &name, const LearningBytes &sealed, LearningBytes &plain, LearningError &error) override {
        if (sealed.size() < 36 || sealed.size() > maximumLearningSnapshot + 36 || std::memcmp(sealed.data(), "ARIONUD1", 8))
            return failed(error, 4, "snapshot envelope");
        if (!cipher_) return failed(error, 1, "key unavailable");
        if (!cipher_->open(labelPrefix_ + name, LearningBytes(sealed.begin() + 8, sealed.end()), plain))
            return failed(error, 2, "AES-GCM authentication");
        return true;
    }
};
} // namespace

std::shared_ptr<LearningProvider> makePosixLearningProvider(const std::string &directory,
    std::shared_ptr<LearningCipher> cipher, const std::string &labelPrefix) {
    return std::make_shared<PosixProvider>(directory, std::move(cipher), labelPrefix);
}

bool learningVaultEntryIsData(const std::string &name) {
    auto endsWith = [&](const char *suffix) {
        size_t length = std::strlen(suffix);
        return name.size() >= length && name.compare(name.size() - length, length, suffix) == 0;
    };
    return name != ".learning.lock" && !endsWith(".arion-userdb.lock") && name != ".DS_Store";
}
} // namespace arion::ime
