#include "arion_rime.h"
#include "../../../apps/desktop/ime/shared/learning_posix.h"
#include "../../../apps/desktop/ime/shared/learning_rime.h"
#include "../../../apps/desktop/ime/shared/english.h"
#include "../../../apps/desktop/ime/shared/spelling.h"
#include "../../../apps/desktop/ime/shared/spelling_options.h"
#include "../../../apps/desktop/ime/shared/ranking.h"
#include "../../../apps/desktop/ime/shared/ranking_unicode.h"
#include <rime/candidate.h>
#include <rime/composition.h>
#include <rime/context.h>
#include <rime/menu.h>
#include <rime/service.h>
#include <rime_api.h>
#include <algorithm>
#include <cctype>
#include <cstdlib>
#include <cstring>
#include <fstream>
#include <map>
#include <mutex>
#include <string>
#include <vector>

// librime-predict, compiled into this library; requiring it keeps its
// module in the static link.
void rime_require_module_predict();

using namespace arion::ime;
namespace {
std::mutex engineMutex;
RimeApi *api = nullptr;
bool started = false;
// Syllables of the shipped dictionary, keyed by their 9-key digits.
std::multimap<std::string, std::string> syllablesByDigits;

void report(char *error, size_t capacity, const std::string &reason) {
    if (!error || !capacity) return;
    size_t count = std::min(capacity - 1, reason.size());
    std::memcpy(error, reason.data(), count);
    error[count] = '\0';
}

char digitOf(char letter) {
    static const char *groups[] = {"abc", "def", "ghi", "jkl", "mno", "pqrs", "tuv", "wxyz"};
    for (int i = 0; i < 8; ++i) if (std::strchr(groups[i], letter)) return static_cast<char>('2' + i);
    return 0;
}

bool loadSyllables(const std::string &data) {
    std::ifstream file(data + "/arion_syllables.txt");
    if (!file) return false;
    syllablesByDigits.clear();
    std::string syllable;
    while (std::getline(file, syllable)) {
        if (syllable.empty()) continue;
        std::string digits;
        for (char c : syllable) {
            char digit = digitOf(c);
            if (!digit) return false;
            digits.push_back(digit);
        }
        syllablesByDigits.emplace(digits, syllable);
    }
    return !syllablesByDigits.empty();
}

class CallbackCipher final : public LearningCipher {
    ArionRimeCipher cipher_;
    static bool run(int (*operation)(void *, const char *, const uint8_t *, size_t, uint8_t **, size_t *),
                    void *context, const std::string &label, const LearningBytes &input, LearningBytes &output) {
        uint8_t *bytes = nullptr;
        size_t count = 0;
        if (!operation || !operation(context, label.c_str(), input.data(), input.size(), &bytes, &count)) {
            std::free(bytes);
            return false;
        }
        output.assign(bytes, bytes + count);
        std::free(bytes);
        return true;
    }
public:
    explicit CallbackCipher(const ArionRimeCipher &cipher) : cipher_(cipher) {}
    bool seal(const std::string &label, const LearningBytes &plain, LearningBytes &sealed) override {
        return run(cipher_.seal, cipher_.context, label, plain, sealed);
    }
    bool open(const std::string &label, const LearningBytes &sealed, LearningBytes &plain) override {
        return run(cipher_.open, cipher_.context, label, sealed, plain);
    }
};

const char *labelPrefix = "com.astrafoundry.arion.mobile.userdb.v1/";

struct Session {
    RimeSessionId id = 0;
    std::string input, preedit, commit, handed;
    // The context of the last commit, for ranking; never host text.
    std::string previous;
    // The first page in display order: engine indices, reordered by context.
    std::vector<size_t> order;
    std::vector<std::string> syllables, texts, comments;
    std::vector<const char *> syllablePointers;
};

Session *sessionOf(ArionRimeSession handle) { return reinterpret_cast<Session *>(handle); }

// The schema's page size (menu/page_size): the page context ranking reorders.
constexpr size_t kRankedPage = 9;

// Encrypted context learning, open while the engine runs with a cipher.
std::unique_ptr<LearningDatabase> contextDatabase;

LearningRecords rankingRecords() {
    auto records = contextDatabase->records();
    for (auto it = records.begin(); it != records.end();) {
        if (!it->first.empty() && it->first[0] == '\x01') it = records.erase(it); else ++it;
    }
    return records;
}

// The desktop's context learning: a committed Chinese word after the last
// one strengthens that pair (Windows keyboard_runtime.cpp arion_keyboard_learn).
void learn(Session &session, const std::string &commit) {
    if (contextDatabase && !contextDatabase->disabled() && rankingContainsHan(commit) == 1) {
        auto changes = ContextRanker(rankingRecords()).learn(commit, session.previous);
        if (!changes.empty()) {
            bool ok = contextDatabase->begin();
            for (const auto &[key, value] : changes) if (ok) ok = value ? contextDatabase->update(key, *value) : contextDatabase->erase(key);
            if (ok) ok = contextDatabase->commit(); else contextDatabase->abort();
            if (!ok) reportEncryptedLearningError(contextDatabase->error());
        }
    }
    session.previous = ContextRanker::contextOf(commit);
}

// Collects text the engine committed since the last read.
void drainCommit(Session &session) {
    RIME_STRUCT(RimeCommit, commit);
    if (api->get_commit(session.id, &commit)) {
        if (commit.text) {
            session.commit += commit.text;
            learn(session, commit.text);
        }
        api->free_commit(&commit);
    }
}

// The first page reordered by the previous commit, as the desktop's menu
// does (menu.c): only candidates spanning the same input as the first one
// may move, and the first stays first unless its replacement is eligible.
void rankFirstPage(Session &session) {
    session.order.clear();
    if (session.input.empty() || !contextDatabase || contextDatabase->disabled()) return;
    auto rime = rime::Service::instance().GetSession(session.id);
    rime::Context *context = rime ? rime->context() : nullptr;
    if (!context || context->composition().empty() || !context->composition().back().menu) return;
    auto &menu = context->composition().back().menu;
    std::vector<std::string> texts;
    std::vector<unsigned char> eligible;
    rime::an<rime::Candidate> first;
    for (size_t i = 0; i < kRankedPage; ++i) {
        auto candidate = menu->GetCandidateAt(i);
        if (!candidate) break;
        if (!first) first = candidate;
        texts.push_back(candidate->text());
        eligible.push_back(i == 0 || (first->end() > first->start() && candidate->start() == first->start() && candidate->end() == first->end()));
    }
    if (texts.size() < 2) return;
    auto order = ContextRanker(rankingRecords()).order(texts, eligible.data(), session.previous);
    if (order.size() != texts.size() || !eligible[order[0]]) return;
    session.order = std::move(order);
}

size_t engineIndex(const Session &session, size_t index) {
    return index < session.order.size() ? session.order[index] : index;
}

// Escape ends a composition and, through librime-predict's predictor, hides
// predictions without predicting again until the next commit.
void dismiss(Session &session) {
    api->process_key(session.id, 0xff1b, 0);
}

// The 9-key syllable list: every dictionary syllable spelled by a prefix of
// the first pending digit run, longest first, then the run's first letters.
void computeSyllables(Session &session) {
    session.syllables.clear();
    const std::string &input = session.input;
    auto start = std::find_if(input.begin(), input.end(), [](char c) { return c >= '2' && c <= '9'; });
    if (start == input.end()) return;
    auto end = std::find_if(start, input.end(), [](char c) { return c < '2' || c > '9'; });
    std::string run(start, end);
    for (size_t length = std::min<size_t>(run.size(), 6); length > 0; --length) {
        auto range = syllablesByDigits.equal_range(run.substr(0, length));
        std::vector<std::string> found;
        for (auto it = range.first; it != range.second; ++it) found.push_back(it->second);
        std::sort(found.begin(), found.end());
        for (auto &syllable : found) if (syllable.size() > 1) session.syllables.push_back(syllable);
    }
    static const char *groups[] = {"abc", "def", "ghi", "jkl", "mno", "pqrs", "tuv", "wxyz"};
    for (const char *letter = groups[run[0] - '2']; *letter; ++letter)
        session.syllables.emplace_back(1, *letter);
}

void refresh(Session &session) {
    drainCommit(session);
    const char *input = api->get_input(session.id);
    session.input = input ? input : "";
    session.preedit.clear();
    RIME_STRUCT(RimeContext, context);
    if (api->get_context(session.id, &context)) {
        if (context.composition.preedit) session.preedit = context.composition.preedit;
        api->free_context(&context);
    }
    computeSyllables(session);
    rankFirstPage(session);
}
} // namespace

extern "C" {
int arion_rime_start(const char *data, const char *user, const char *vault,
                     const ArionRimeCipher *cipher, char *error, size_t capacity) {
    std::lock_guard<std::mutex> lock(engineMutex);
    if (started) return 1;
    if (!data || !user || !vault) { report(error, capacity, "missing directory"); return 0; }
    if (!loadSyllables(data)) { report(error, capacity, "syllable list unavailable"); return 0; }
    api = rime_get_api();
    if (!api) { report(error, capacity, "engine unavailable"); return 0; }
    std::string build = std::string(data) + "/build";
    RIME_STRUCT(RimeTraits, traits);
    traits.shared_data_dir = data;
    traits.prebuilt_data_dir = build.c_str();
    traits.staging_dir = build.c_str();
    traits.user_data_dir = user;
    traits.app_name = "rime.arion";
    traits.distribution_name = "Arion";
    traits.distribution_code_name = "arion";
    traits.distribution_version = "1";
    traits.min_log_level = 3;
    traits.log_dir = "";
    // librime's default modules plus librime-predict.
    rime_require_module_predict();
    static const char *modules[] = {"default", "predict", nullptr};
    traits.modules = modules;
    api->setup(&traits);
    if (cipher) {
        LearningError failure;
        auto provider = makePosixLearningProvider(vault, std::make_shared<CallbackCipher>(*cipher), labelPrefix);
        if (!registerEncryptedLearning(provider, failure)) {
            report(error, capacity, "learning: " + failure.operation);
            return 0;
        }
    }
    api->initialize(&traits);
    // The desktop's correcting Pinyin translator and its bilingual wrapper.
    arion_spelling_register();
    char english[256] = {0};
    if (!arion_english_register(data, english, sizeof(english))) {
        api->finalize();
        arion_english_unregister();
        auto cause = encryptedLearningError();
        report(error, capacity, std::string("english: ") + english + (cause.code ? " (" + cause.operation + ")" : ""));
        return 0;
    }
    if (cipher) {
        LearningError issue;
        contextDatabase = openContextLearning(issue);
        if (!contextDatabase) {
            api->finalize();
            arion_english_unregister();
            report(error, capacity, "learning: " + issue.operation);
            return 0;
        }
    }
    auto failure = encryptedLearningError();
    if (failure.code) {
        api->finalize();
        report(error, capacity, "learning: " + failure.operation);
        return 0;
    }
    started = true;
    return 1;
}

int arion_rime_reset_learning(const char *vault, char *error, size_t capacity) {
    LearningError failure;
    if (!resetLearningVault(makePosixLearningProvider(vault, nullptr, labelPrefix), failure)) {
        report(error, capacity, "learning: " + failure.operation);
        return 0;
    }
    clearEncryptedLearningError();
    return 1;
}

void arion_rime_stop(void) {
    std::lock_guard<std::mutex> lock(engineMutex);
    if (!started) return;
    api->finalize();
    arion_english_unregister();
    contextDatabase.reset();
    started = false;
}

ArionRimeSession arion_rime_open(const char *schema) {
    if (!started || !schema) return 0;
    auto session = new Session;
    session->id = api->create_session();
    if (!session->id || !api->select_schema(session->id, schema)) {
        if (session->id) api->destroy_session(session->id);
        delete session;
        return 0;
    }
    // No predictions before anything is typed.
    dismiss(*session);
    refresh(*session);
    session->commit.clear();
    return reinterpret_cast<ArionRimeSession>(session);
}

void arion_rime_close(ArionRimeSession handle) {
    Session *session = sessionOf(handle);
    if (!session) return;
    api->destroy_session(session->id);
    delete session;
}

int arion_rime_type(ArionRimeSession handle, char key) {
    Session *session = sessionOf(handle);
    if (!session) return 0;
    session->commit.clear();
    int handled = api->process_key(session->id, static_cast<unsigned char>(key), 0);
    refresh(*session);
    return handled;
}

int arion_rime_backspace(ArionRimeSession handle) {
    Session *session = sessionOf(handle);
    if (!session) return 0;
    session->commit.clear();
    int handled = api->process_key(session->id, 0xff08, 0);
    refresh(*session);
    return handled;
}

int arion_rime_set_input(ArionRimeSession handle, const char *input) {
    Session *session = sessionOf(handle);
    if (!session || !input) return 0;
    session->commit.clear();
    int ok = api->set_input(session->id, input);
    refresh(*session);
    return ok;
}

int arion_rime_select(ArionRimeSession handle, size_t index) {
    Session *session = sessionOf(handle);
    if (!session) return 0;
    session->commit.clear();
    int ok = api->select_candidate(session->id, engineIndex(*session, index));
    refresh(*session);
    return ok;
}

int arion_rime_commit_raw(ArionRimeSession handle) {
    Session *session = sessionOf(handle);
    if (!session) return 0;
    const char *input = api->get_input(session->id);
    std::string raw = input ? input : "";
    api->clear_composition(session->id);
    refresh(*session);
    session->commit = raw;
    return !raw.empty();
}

void arion_rime_clear(ArionRimeSession handle) {
    Session *session = sessionOf(handle);
    if (!session) return;
    api->clear_composition(session->id);
    dismiss(*session);
    refresh(*session);
    session->commit.clear();
    // The field changed: what came before is no longer known.
    session->previous.clear();
}

void arion_rime_dismiss(ArionRimeSession handle) {
    Session *session = sessionOf(handle);
    if (!session) return;
    dismiss(*session);
    refresh(*session);
    session->commit.clear();
}

void arion_rime_options(ArionRimeSession handle, int typo, uint32_t fuzzy, int prediction) {
    Session *session = sessionOf(handle);
    if (!session) return;
    arion_spelling_apply(api, session->id, typo, fuzzy & ARION_FUZZY_MASK);
    api->set_option(session->id, "prediction", prediction ? True : False);
    if (!prediction) dismiss(*session);
    refresh(*session);
    session->commit.clear();
}

int arion_rime_state(ArionRimeSession handle, ArionRimeState *state) {
    Session *session = sessionOf(handle);
    if (!session || !state) return 0;
    session->syllablePointers.clear();
    for (const auto &syllable : session->syllables) session->syllablePointers.push_back(syllable.c_str());
    state->input = session->input.c_str();
    state->preedit = session->preedit.c_str();
    // A commit is handed out once, so a second read cannot insert it again.
    session->handed = std::move(session->commit);
    session->commit.clear();
    state->commit = session->handed.empty() ? nullptr : session->handed.c_str();
    state->syllables = session->syllablePointers.data();
    state->syllableCount = session->syllablePointers.size();
    return 1;
}

size_t arion_rime_candidates(ArionRimeSession handle, size_t offset, size_t limit,
                             const char **texts, const char **comments) {
    Session *session = sessionOf(handle);
    if (!session || !limit) return 0;
    session->texts.clear();
    session->comments.clear();
    // The ranked first page is read from the start and shown in display order.
    const size_t from = offset < session->order.size() ? 0 : offset;
    const size_t want = offset < session->order.size() ? std::max(offset + limit, session->order.size()) : limit;
    std::vector<std::string> engineTexts, engineComments;
    RimeCandidateListIterator iterator = {};
    if (!api->candidate_list_from_index(session->id, &iterator, static_cast<int>(from))) return 0;
    while (engineTexts.size() < want && api->candidate_list_next(&iterator)) {
        engineTexts.emplace_back(iterator.candidate.text ? iterator.candidate.text : "");
        engineComments.emplace_back(iterator.candidate.comment ? iterator.candidate.comment : "");
    }
    api->candidate_list_end(&iterator);
    for (size_t display = offset; display < offset + limit; ++display) {
        const size_t engine = (from == 0 ? engineIndex(*session, display) : display) - from;
        if (engine >= engineTexts.size()) break;
        session->texts.push_back(engineTexts[engine]);
        session->comments.push_back(engineComments[engine]);
    }
    for (size_t i = 0; i < session->texts.size(); ++i) {
        if (texts) texts[i] = session->texts[i].c_str();
        if (comments) comments[i] = session->comments[i].c_str();
    }
    return session->texts.size();
}

size_t arion_rime_english(const char *word, size_t limit, const char **texts) {
    std::lock_guard<std::mutex> lock(engineMutex);
    return started ? arion_english_candidates(word, limit, texts) : 0;
}

int arion_rime_english_confirm(const char *word) {
    std::lock_guard<std::mutex> lock(engineMutex);
    return started ? arion_english_confirm(word) : 0;
}
}
