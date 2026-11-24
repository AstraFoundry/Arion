#include "learning_posix.h"
#include <cstdlib>
#include <filesystem>
#include <fstream>
#include <iostream>
using namespace arion::ime;
static void require(bool ok, const char *message) {
    if (!ok) { std::cerr << "POSIX learning test failed: " << message << '\n'; std::exit(1); }
}
// A deterministic authenticated stand-in: the label and a checksum guard the
// bytes, so a wrong label or tampering fails like AES-GCM would.
class TestCipher final : public LearningCipher {
    static uint32_t sum(const std::string &label, const LearningBytes &bytes) {
        uint32_t value = 2166136261u;
        for (char c : label) value = (value ^ static_cast<uint8_t>(c)) * 16777619u;
        for (uint8_t b : bytes) value = (value ^ b) * 16777619u;
        return value;
    }
public:
    bool seal(const std::string &label, const LearningBytes &plain, LearningBytes &sealed) override {
        sealed.assign(28, 0);
        uint32_t check = sum(label, plain);
        for (int i = 0; i < 4; ++i) sealed[i] = static_cast<uint8_t>(check >> (8 * i));
        for (uint8_t b : plain) sealed.push_back(b ^ 0x5a);
        return true;
    }
    bool open(const std::string &label, const LearningBytes &sealed, LearningBytes &plain) override {
        if (sealed.size() < 28) return false;
        LearningBytes next;
        for (size_t i = 28; i < sealed.size(); ++i) next.push_back(sealed[i] ^ 0x5a);
        uint32_t check = sum(label, next);
        for (int i = 0; i < 4; ++i) if (sealed[i] != static_cast<uint8_t>(check >> (8 * i))) return false;
        plain = std::move(next);
        return true;
    }
};
int main(int argc, char **argv) {
    require(argc == 2, "vault directory required");
    const std::string vault = argv[1];
    std::filesystem::create_directories(vault);
    auto provider = makePosixLearningProvider(vault, std::make_shared<TestCipher>(), "test.userdb.v1/");
    const std::string file = learningPath(vault, "arion_pinyin.userdb.arion-userdb");
    {
        LearningDatabase db(provider, file, "arion_pinyin");
        require(db.open(), "open a new vault");
        require(db.wasMissing(), "new snapshot reported missing");
        require(db.begin() && db.update("ni hao", "c=1") && db.update(std::string("\0k", 2), "v") && db.commit(), "commit learning");
        require(db.close(), "close");
    }
    {
        LearningDatabase db(provider, file, "arion_pinyin");
        std::string value;
        require(db.open(true) && db.fetch("ni hao", &value) && value == "c=1", "reopen and read back");
        require(db.fetch(std::string("\0k", 2), &value) && value == "v", "embedded NUL key survives");
        require(db.close(), "close readonly");
    }
    std::ifstream stored(file, std::ios::binary);
    std::string head(8, '\0');
    stored.read(head.data(), 8);
    require(head == "ARIONUD1", "snapshot carries the ARIONUD1 envelope");
    LearningError error;
    LearningBytes sealed, plain;
    bool missing = false;
    require(provider->read(file, sealed, missing, error) && !missing, "read sealed snapshot");
    require(!provider->unseal("other_name", sealed, plain, error) && error.code == 2, "label binds the database name");
    auto tampered = sealed;
    tampered.back() ^= 1;
    require(!provider->unseal("arion_pinyin", tampered, plain, error) && error.code == 2, "tampering is rejected");
    tampered = sealed;
    tampered[0] = 'X';
    require(!provider->unseal("arion_pinyin", tampered, plain, error) && error.code == 4, "envelope is checked first");
    auto keyless = makePosixLearningProvider(vault, nullptr, "test.userdb.v1/");
    require(!keyless->unseal("arion_pinyin", sealed, plain, error) && error.code == 1, "no cipher means no plaintext");
    require(!keyless->seal("arion_pinyin", plain, sealed, error) && error.code == 1, "no cipher cannot write");
    std::vector<std::string> names;
    require(provider->files(names, error), "enumerate vault");
    bool sawSnapshot = false;
    for (const auto &name : names) {
        require(name != "." && name != "..", "enumeration skips dot entries");
        require(name.rfind(".encrypted-", 0) != 0, "no temporary file left behind");
        if (name == "arion_pinyin.userdb.arion-userdb") sawSnapshot = true;
    }
    require(sawSnapshot, "snapshot listed");
    require(learningVaultEntryIsData("arion_pinyin.userdb.arion-userdb") && learningVaultEntryIsData(".encrypted-abc123"),
            "snapshots and interrupted writes count as data");
    require(!learningVaultEntryIsData(".learning.lock") && !learningVaultEntryIsData("x.arion-userdb.lock") &&
            !learningVaultEntryIsData(".DS_Store"), "locks and Finder metadata do not");
    std::unique_ptr<LearningLock> first, second;
    require(provider->lock(file, false, false, first, error), "writer lease");
    require(!provider->lock(file, false, false, second, error) && error.code == 8, "a second writer is refused");
    first.reset();
    require(provider->lock(file, false, false, second, error), "lease released");
    std::cout << "POSIX learning provider passed.\n";
    return 0;
}
