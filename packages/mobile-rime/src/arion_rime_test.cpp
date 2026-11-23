// Host check of the phone bridge against the staged data (ADR 053).
#include "arion_rime.h"
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <iostream>
#include <string>
#include <vector>

static void require(bool ok, const std::string &message) {
    if (!ok) { std::cerr << "Mobile Rime test failed: " << message << '\n'; std::exit(1); }
}
// Reversible, authenticated stand-in for the OS cipher.
static int seal(void *, const char *label, const uint8_t *plain, size_t size, uint8_t **out, size_t *outSize) {
    size_t tag = std::strlen(label);
    *out = static_cast<uint8_t *>(std::malloc(28 + size));
    std::memset(*out, 0, 28);
    std::memcpy(*out, &tag, sizeof tag);
    for (size_t i = 0; i < size; ++i) (*out)[28 + i] = plain[i] ^ 0x3c;
    *outSize = 28 + size;
    return 1;
}
static int open(void *, const char *label, const uint8_t *sealed, size_t size, uint8_t **out, size_t *outSize) {
    size_t tag = 0;
    if (size < 28) return 0;
    std::memcpy(&tag, sealed, sizeof tag);
    if (tag != std::strlen(label)) return 0;
    *out = static_cast<uint8_t *>(std::malloc(size - 28 + 1));
    for (size_t i = 28; i < size; ++i) (*out)[i - 28] = sealed[i] ^ 0x3c;
    *outSize = size - 28;
    return 1;
}
static std::vector<std::string> candidates(ArionRimeSession session, size_t limit = 30) {
    std::vector<const char *> texts(limit);
    size_t count = arion_rime_candidates(session, 0, limit, texts.data(), nullptr);
    return {texts.begin(), texts.begin() + static_cast<long>(count)};
}
static long indexOf(const std::vector<std::string> &list, const std::string &text) {
    for (size_t i = 0; i < list.size(); ++i) if (list[i] == text) return static_cast<long>(i);
    return -1;
}
static void type(ArionRimeSession session, const std::string &keys) {
    for (char key : keys) require(arion_rime_type(session, key), std::string("key ") + key);
}
static std::string commitOf(ArionRimeSession session) {
    ArionRimeState state{};
    require(arion_rime_state(session, &state), "state");
    return state.commit ? state.commit : "";
}
static void start(const char *data, const std::string &user, const std::string &vault) {
    ArionRimeCipher cipher{nullptr, seal, open};
    char error[256] = {0};
    require(arion_rime_start(data, user.c_str(), vault.c_str(), &cipher, error, sizeof error), std::string("start: ") + error);
}

int main(int argc, char **argv) {
    require(argc == 3, "data and scratch directories required");
    const char *data = argv[1];
    const std::string scratch = argv[2], user = scratch + "/user", vault = user + "/learning";
    std::filesystem::create_directories(vault);
    start(data, user, vault);

    ArionRimeSession full = arion_rime_open("arion_pinyin");
    require(full, "open arion_pinyin");
    type(full, "nihao");
    auto list = candidates(full);
    require(indexOf(list, "你好") == 0, "nihao leads with 你好");
    ArionRimeState state{};
    require(arion_rime_state(full, &state) && std::string(state.input) == "nihao", "raw input kept");
    require(arion_rime_select(full, 0) && commitOf(full) == "你好", "selecting commits");
    require(arion_rime_state(full, &state) && std::string(state.input).empty(), "composition ends after commit");
    require(commitOf(full).empty(), "a commit is handed out once");
    type(full, "wifi");
    require(arion_rime_commit_raw(full) && commitOf(full) == "wifi", "return commits the raw letters");
    type(full, "ni");
    require(arion_rime_backspace(full) && arion_rime_state(full, &state) && std::string(state.input) == "n", "backspace removes one key");
    arion_rime_clear(full);
    require(arion_rime_state(full, &state) && std::string(state.input).empty(), "clear drops the composition");

    // English, from the desktop's lexicon: completions in the English layout,
    // and familiar words mixed into Pinyin as on the desktop.
    const char *english[30] = {};
    std::vector<std::string> words;
    for (size_t i = 0, n = arion_rime_english("hel", 30, english); i < n; ++i) words.push_back(english[i]);
    require(indexOf(words, "help") >= 0 && indexOf(words, "hello") >= 0, "hel completes to help and hello");
    require(indexOf(words, "hello") < indexOf(words, "Hel") || indexOf(words, "Hel") < 0, "completions come before a rare exact form");
    words.clear();
    for (size_t i = 0, n = arion_rime_english("Hel", 30, english); i < n; ++i) words.push_back(english[i]);
    require(indexOf(words, "Hello") >= 0, "completions follow the typed capital");
    require(arion_rime_english_confirm("hello"), "a chosen English word is recorded");
    type(full, "hello");
    require(indexOf(candidates(full), "hello") >= 0, "hello is offered in Pinyin");
    arion_rime_clear(full);
    type(full, "xin");
    require(indexOf(candidates(full), "新") >= 0 && indexOf(candidates(full), "新") < 3, "xin keeps Chinese first");
    arion_rime_clear(full);

    ArionRimeSession keypad = arion_rime_open("arion_pinyin_9key");
    require(keypad, "open arion_pinyin_9key");
    type(keypad, "64426");
    require(indexOf(candidates(keypad), "你好") >= 0 && indexOf(candidates(keypad), "你好") < 3, "64426 offers 你好 near the front");
    require(arion_rime_state(keypad, &state) && state.syllableCount > 0, "9-key lists syllables");
    std::vector<std::string> syllables(state.syllables, state.syllables + state.syllableCount);
    require(indexOf(syllables, "ni") >= 0 && indexOf(syllables, "mi") >= 0, "64 spells ni and mi");
    require(syllables.back() == "o" && indexOf(syllables, "m") >= 0, "the first digit's letters close the list");
    require(arion_rime_set_input(keypad, "mi426") && indexOf(candidates(keypad), "你好") < 0, "a chosen syllable constrains the digits");
    arion_rime_clear(keypad);

    // Learning: a later candidate chosen twice moves forward after a restart.
    type(full, "shi");
    auto before = candidates(full, 60);
    long chosen = 8;
    require(static_cast<long>(before.size()) > chosen, "enough shi candidates");
    const std::string learned = before[chosen];
    require(arion_rime_select(full, chosen) && commitOf(full) == learned, "commit a later candidate");
    type(full, "shi");
    require(arion_rime_select(full, indexOf(candidates(full, 60), learned)), "commit it again");
    arion_rime_close(full);
    arion_rime_close(keypad);
    arion_rime_stop();
    bool sealed = false;
    for (const auto &entry : std::filesystem::directory_iterator(vault)) {
        std::ifstream file(entry.path(), std::ios::binary);
        char head[8] = {0};
        file.read(head, 8);
        if (std::string(head, 8) == "ARIONUD1") sealed = true;
    }
    require(sealed, "learning is stored in the ARIONUD1 envelope");
    start(data, user, vault);
    full = arion_rime_open("arion_pinyin");
    type(full, "shi");
    long after = indexOf(candidates(full, 60), learned);
    require(after >= 0 && after < chosen, "learning survives a restart");
    arion_rime_close(full);
    arion_rime_stop();
    char error[256] = {0};
    require(arion_rime_reset_learning(vault.c_str(), error, sizeof error), std::string("reset: ") + error);
    std::cout << "Mobile Rime bridge passed: 26-key, 9-key, syllables, raw commit, English and encrypted learning.\n";
    return 0;
}
