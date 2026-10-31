#import "typing.h"
#include "../shared/input.h"
#include "../shared/ranking_bridge.h"
#include "../shared/emoji.h"
#include "../shared/completion.h"
#include "../shared/spelling_options.h"
#include "../shared/personal_vocabulary_bridge.h"
#import <InputMethodKit/InputMethodKit.h>
#include <ctype.h>

// A previous commit older than this no longer predicts the next word.
static const NSTimeInterval kContextLifetime = 120;

static BOOL containsHan(NSString *text) {
    return arion_ranker_contains_han(text.UTF8String,[text lengthOfBytesUsingEncoding:NSUTF8StringEncoding])==1;
}

@interface ArionTyping ()
@property(strong) ArionKeyboardEngine *engine;
@property(weak) id client;
@property uintptr_t session;
@property NSUInteger engineGeneration;
@property(readwrite) BOOL composing;
@property(readwrite) BOOL ascii;
@property(readwrite) BOOL available;
@property(readwrite, copy) NSString *recoveryText;
@property(readwrite) NSUInteger revision;
@property(readwrite, strong) ArionKeyboardSnapshot *snapshot;
// The engine snapshot the visible page was derived from.
@property(strong) ArionKeyboardSnapshot *engineSnapshot;
@property ArionPinyinMenu menu;
@property(copy) NSString *previousCommit;
@property NSTimeInterval previousCommitTime;
@property NSEventModifierFlags lastModifierFlags;
@property ArionShiftTap shiftTap;
@property BOOL emojiInComposition;
@property(copy) NSArray<NSDictionary *> *rankingCandidates;
@property(copy) NSArray<NSNumber *> *rankingSlots;
@property NSUInteger rankingRevision;
@end

static int sharedMenuMetadata(void *context, uintptr_t session, size_t index, ArionCandidateMetadata *metadata) {
    (void)context;
    return arion_candidate_metadata(session,index,metadata);
}

static int rankSharedMenu(void *context, const char *const *texts, size_t count, const unsigned char *eligible, size_t *order) {
    NSDictionary *ranking = (__bridge NSDictionary *)context;
    NSMutableArray *candidates = [NSMutableArray arrayWithCapacity:count];
    for (size_t i = 0; i < count; i++) {
        NSString *text = [[NSString alloc] initWithUTF8String:texts[i]];
        if (!text) return 0;
        [candidates addObject:text];
    }
    NSArray<NSNumber *> *indices = [ranking[@"ranker"] order:candidates eligible:eligible after:ranking[@"previous"]];
    if (indices.count != count) return 0;
    for (size_t i = 0; i < count; i++) order[i] = indices[i].unsignedIntegerValue;
    return 1;
}

static NSArray<NSString *> *sharedMenuTexts(const ArionPinyinMenu *menu, NSUInteger start, NSUInteger count) {
    NSMutableArray *texts = [NSMutableArray arrayWithCapacity:count];
    for (NSUInteger i = start; i < start + count; i++)
        [texts addObject:[[NSString alloc] initWithUTF8String:menu->texts[i]]];
    return texts;
}
@implementation ArionTyping
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine {
    if ((self = [super init])) {
        self.engine = engine;
        self.pageSize = 9;
        self.contextRanking = YES;
        self.emojiSuggestions = YES;
    }
    return self;
}
- (void)setPageSize:(NSUInteger)pageSize {
    _pageSize = MAX(1, MIN(9, pageSize));
}
- (void)setTypoCorrection:(BOOL)enabled {
    if (_typoCorrection == enabled) return;
    [self cancel];
    _typoCorrection = enabled;
    if (self.engineGeneration == self.engine.generation)
        arion_spelling_apply(self.engine.api,self.session,_typoCorrection,_fuzzyPinyin);
}
- (void)setFuzzyPinyin:(uint32_t)pairs {
    pairs &= ARION_FUZZY_MASK;
    if (_fuzzyPinyin == pairs) return;
    [self cancel];
    _fuzzyPinyin = pairs;
    if (self.engineGeneration == self.engine.generation)
        arion_spelling_apply(self.engine.api,self.session,_typoCorrection,_fuzzyPinyin);
}
- (void)dealloc {
    arion_menu_dispose(&_menu);
    // Engine sessions outlive their Objective-C owner unless explicitly closed.
    // Destruction must not call into a disappearing client or its UI callbacks.
    if (_session && _engineGeneration == _engine.generation) {
        [_engine clear:_session];
        [_engine destroySession:_session];
    }
}
- (BOOL)activate:(id)client error:(NSError **)error {
    [self deactivate];
    self.client = client;
    self.recoveryText = @"";
    self.session = [self.engine createSession];
    self.engineGeneration = self.engine.generation;
    if (!self.session) {
        if (error) *error = [NSError errorWithDomain:@"ArionKeyboard" code:7 userInfo:nil];
        return NO;
    }
    [self.engine setEmojiSuggestions:self.emojiSuggestions session:self.session];
    arion_spelling_apply(self.engine.api,self.session,self.typoCorrection,self.fuzzyPinyin);
    self.available = [self refresh:error];
    return self.available;
}
- (void)deactivate {
    [self cancel];
    if (self.engineGeneration == self.engine.generation) [self.engine destroySession:self.session];
    self.session = 0;
    self.available = NO;
    self.client = nil;
    self.previousCommit = nil;
}
- (void)cancel {
    [self freezeRanking];
    arion_shift_reset(&_shiftTap);
    if (self.engineGeneration == self.engine.generation) [self.engine clear:self.session];
    BOOL clear = self.composing;
    self.composing = NO;
    _menu.expanded = NO;
    _menu.top_row = 0;
    self.snapshot = nil;
    self.engineSnapshot = nil;
    self.emojiInComposition = NO;
    arion_menu_dispose(&_menu);

    self.revision++;
    if (clear) [self.client setMarkedText:@"" selectionRange:NSMakeRange(0, 0)
                       replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    if (self.changed) self.changed();
}
- (void)toggleASCII {
    if (!self.session || !self.available || self.engineGeneration != self.engine.generation) return;
    [self cancel];
    self.previousCommit = nil;
    [self.engine setASCII:!self.ascii session:self.session];
    if ([self refresh:nil] && self.modeChanged) self.modeChanged(ArionTypingSwitchLanguage);
}
- (BOOL)fullWidth {
    return self.engine.fullWidth;
}
- (BOOL)englishPunctuation {
    return self.engine.englishPunctuation;
}
- (BOOL)toggleFullWidth {
    if (!self.live) return NO;
    [self cancel];
    self.previousCommit = nil;
    if (![self.engine setFullWidth:!self.engine.fullWidth session:self.session]) return NO;
    if (self.modeChanged) self.modeChanged(ArionTypingSwitchWidth);
    return YES;
}
- (BOOL)toggleEnglishPunctuation {
    if (!self.live) return NO;
    [self cancel];
    self.previousCommit = nil;
    if (![self.engine setEnglishPunctuation:!self.engine.englishPunctuation session:self.session]) return NO;
    if (self.modeChanged) self.modeChanged(ArionTypingSwitchPunctuation);
    return YES;
}
- (BOOL)live {
    return self.session && self.available && self.engineGeneration == self.engine.generation;
}

// JSON integers must be lossless. NSNumber also represents booleans and floating
// values, which must not be accepted as request or persisted revision identities.
static BOOL vocabularyInteger(id value, uint64_t *result) {
    if (![value isKindOfClass:NSNumber.class] || CFGetTypeID((__bridge CFTypeRef)value)==CFBooleanGetTypeID()) return NO;
    const char *kind=[value objCType];
    if (!kind || !strchr("cCsSiIlLqQ",kind[0]) || kind[1]) return NO;
    NSString *text=[value stringValue];
    uint64_t number=[value unsignedLongLongValue];
    if (![text isEqual:[NSString stringWithFormat:@"%llu",(unsigned long long)number]]) return NO;
    *result=number;
    return YES;
}

static const char *vocabularyString(id value, NSUInteger maximum) {
    if (![value isKindOfClass:NSString.class]) return NULL;
    NSUInteger bytes=[value lengthOfBytesUsingEncoding:NSUTF8StringEncoding];
    if (!bytes || bytes>maximum) return NULL;
    const char *text=[value UTF8String];
    return text && strlen(text)==bytes ? text : NULL;
}

- (NSDictionary *)applyVocabularyMessage:(NSDictionary *)message activation:(NSString *)activation blocked:(BOOL)blocked {
    if (![message isKindOfClass:NSDictionary.class]) return nil;
    BOOL query=[message[@"op"] isEqual:@"vocabulary_query"];
    if (!query && ![message[@"op"] isEqual:@"vocabulary_publish"]) return nil;
    uint64_t request=0;
    if (!NSThread.isMainThread || !activation.length || ![message[@"activation"] isEqual:activation] ||
        !vocabularyInteger(message[@"request"],&request) || !request) return nil;
    NSMutableDictionary *reply=[@{@"op":query?@"vocabulary_state":@"vocabulary_published",
        @"activation":activation,@"request":@(request),@"status":@"rejected"} mutableCopy];
    if (!self.live) return reply;
    if (blocked || self.composing) { reply[@"status"]=@"busy"; return reply; }
    uint64_t revision=0;
    if (query) {
        if (arion_personal_vocabulary_restore(self.session,&revision)) {
            reply[@"status"]=@"ok";
            reply[@"vocabularyRevision"]=@(revision);
        }
        return reply;
    }
    id entries=message[@"terms"];
    if (!vocabularyInteger(message[@"vocabularyRevision"],&revision) || !revision ||
        ![entries isKindOfClass:NSArray.class] || [entries count]>2000) return reply;
    NSUInteger count=[entries count];
    ArionPersonalTerm *terms=count?calloc(count,sizeof(*terms)):NULL;
    if (count && !terms) return reply;
    BOOL valid=YES;
    for (NSUInteger i=0;i<count;++i) {
        id entry=entries[i];
        if (![entry isKindOfClass:NSDictionary.class]) { valid=NO; break; }
        terms[i]=(ArionPersonalTerm){vocabularyString(entry[@"id"],128),
            vocabularyString(entry[@"text"],256),vocabularyString(entry[@"reading"],448)};
        if (!terms[i].id || !terms[i].text || !terms[i].reading) { valid=NO; break; }
    }
    if (valid && arion_personal_vocabulary_publish(self.session,revision,terms,count)) {
        reply[@"status"]=@"ok";
        reply[@"vocabularyRevision"]=@(revision);
    }
    free(terms);
    return reply;
}
- (BOOL)handleEvent:(NSEvent *)event {
    if (!self.live) return NO;
    NSEventModifierFlags flags = event.modifierFlags;
    NSEventModifierFlags previousFlags = self.lastModifierFlags;
    NSEventModifierFlags changes = (flags ^ previousFlags) & NSEventModifierFlagDeviceIndependentFlagsMask;
    self.lastModifierFlags = flags;
    if (event.type == NSEventTypeFlagsChanged) {
        BOOL shift = (flags & NSEventModifierFlagShift) != 0;
        unsigned short keyCode = event.keyCode;
        // Some IMK clients omit the modifier keycode. Recognize Shift only
        // when the flag transition identifies that modifier unambiguously.
        BOOL anonymous = keyCode == 0 || keyCode == 255;
        BOOL shiftKey = keyCode == 56 || keyCode == 60 || (anonymous && changes == NSEventModifierFlagShift);
        BOOL otherModifiers = (flags & (NSEventModifierFlagCommand | NSEventModifierFlagControl |
                                      NSEventModifierFlagOption | NSEventModifierFlagCapsLock | NSEventModifierFlagFunction)) != 0;
        BOOL toggle = arion_shift_transition(&_shiftTap, shiftKey, shift, anonymous ? 0 : keyCode,
            otherModifiers, flags == previousFlags, event.timestamp);
        if (toggle) {
            if (!self.ascii && self.composing) {
                uintptr_t session = self.session;
                if (!arion_pinyin_commit_raw(self.engine.api, session)) { NSBeep(); return YES; }
                if (![self refresh:nil] || !self.live || self.session != session) return YES;
            }
            [self toggleASCII];
        }
        return toggle;
    }
    if (event.type != NSEventTypeKeyDown) return NO;
    arion_shift_interrupt(&_shiftTap);
    int policyKey = event.keyCode == 49 ? ' ' : [event.charactersIgnoringModifiers isEqual:@"."] ? '.' : ArionKeyboardKey(event);
    ArionKeyAction action = arion_key_action(policyKey, ArionKeyboardModifiers(event), self.ascii);
    if (action == ARION_KEY_TOGGLE_ASCII) {
        [self toggleASCII];
        return YES;
    }
    // Ctrl+. switches Chinese-mode punctuation; English mode passes it through.
    if (action == ARION_KEY_TOGGLE_PUNCTUATION) {
        if (![self toggleEnglishPunctuation]) NSBeep();
        return YES;
    }
    if (action == ARION_KEY_PASS) {
        [self cancel];
        self.previousCommit = nil;
        return NO;
    }
    if (self.composing && _menu.count > 0 && !self.ascii) {
        NSInteger handled = [self menuKey:event];
        if (handled >= 0) return handled;
    }
    int key = arion_key_normalize(ArionKeyboardKey(event), ArionKeyboardModifiers(event), self.ascii);
    if (key < 0) { [self cancel]; self.previousCommit = nil; return NO; }
    if (self.composing && _menu.highlight < _menu.count && _menu.metadata[_menu.highlight].emoji &&
        key < 128 && ispunct(key)) self.emojiInComposition = YES;
    BOOL wasComposing = self.composing;
    BOOL handled = [self.engine processKey:key modifiers:ArionKeyboardModifiers(event) session:self.session];
    if (wasComposing && self.emojiSuggestions &&
        (event.keyCode == 115 || event.keyCode == 119 || event.keyCode == 123 || event.keyCode == 124) &&
        !arion_emoji_refresh_after_caret_move(self.session)) {
        self.available = NO;
        [self cancel];
        return YES;
    }
    if (![self refresh:nil]) {
        NSBeep();
        return YES;
    }
    if (handled && ((key >= 'a' && key <= 'z') || (key >= 'A' && key <= 'Z') || key == '\'' || key == 0xff08 || key == 0xffff)) [self requestRanking];
    // A key the host handles (Return, arrows, Delete) may move its caret away
    // from the last commit, which then stops being context.
    if (!handled && !wasComposing) self.previousCommit = nil;
    return handled;
}
// Keys that act on Arion's menu. Returns -1 for keys the engine handles.
- (NSInteger)menuKey:(NSEvent *)event {
    NSString *characters = event.characters;
    unichar character = characters.length == 1 ? [characters characterAtIndex:0] : 0;
    NSUInteger first = _menu.page * _menu.page_size;
    if (character >= '1' && character <= '9') {
        NSUInteger index = first + (NSUInteger)(character - '1');
        // A number past the page's end selects nothing but stays a selection key.
        if (character - '1' < (int)_menu.page_size && index < _menu.count) [self choose:index];
        return YES;
    }
    if (character == ' ') { [self choose:_menu.highlight]; return YES; }
    BOOL previous = character == '-' || character == '[' || event.keyCode == 116 || event.keyCode == 126;
    BOOL next = character == '=' || character == ']' || event.keyCode == 121 || event.keyCode == 125;
    if (_menu.expanded) {
        // Rows are pages: move between them; above the first row collapses.
        if (previous) { if (_menu.page == 0) [self setGrid:NO]; else [self turn:-1]; return YES; }
        if (next) { [self turn:1]; return YES; }
        if (event.keyCode == 123 || event.keyCode == 124) { [self moveInRow:event.keyCode == 123 ? -1 : 1]; return YES; }
        return -1;
    }
    if (previous) { [self turn:-1]; return YES; }
    // The next page opens the grid with that page as the current row, as
    // Chatterfly's spread list; a single page has nothing to expand.
    if (next) {
        if ([self verify] && [self load:(_menu.page + 1) * _menu.page_size] && (_menu.page + 1) * _menu.page_size < _menu.count) {
            _menu.expanded = YES;
            _menu.top_row = _menu.page;
            if (![self turn:1]) _menu.expanded = NO;
        }
        return YES;
    }
    return -1;
}
- (BOOL)choose:(NSUInteger)index {
    if (index >= _menu.count) return NO;
    BOOL emoji = _menu.metadata[index].emoji != 0;
    if (![self.engine selectCandidate:_menu.indices[index] session:self.session]) return NO;
    self.emojiInComposition |= emoji;
    return [self refresh:nil];
}
- (BOOL)select:(NSUInteger)index revision:(NSUInteger)revision {
    if (!self.live || revision != self.revision || index >= self.snapshot.candidates.count) return NO;
    return [self choose:_menu.page * _menu.page_size + index];
}
- (BOOL)highlight:(NSUInteger)index revision:(NSUInteger)revision {
    if (!self.live || revision != self.revision || index >= self.snapshot.candidates.count || ![self verify]) return NO;
    _menu.highlight = _menu.page * _menu.page_size + index;
    return [self publishMenu];
}
- (BOOL)nextPage:(NSUInteger)revision {
    if (!self.live || revision != self.revision || !self.composing) return NO;
    return [self turn:1];
}
- (BOOL)select:(NSUInteger)index row:(NSUInteger)row revision:(NSUInteger)revision {
    if (!self.live || revision != self.revision || row >= self.snapshot.rows.count || index >= self.snapshot.rows[row].count) return NO;
    return [self choose:(_menu.top_row + row) * _menu.page_size + index];
}
- (BOOL)toggleExpanded:(NSUInteger)revision {
    if (!self.live || revision != self.revision || !self.composing || ![self verify]) return NO;
    if (_menu.expanded) return [self setGrid:NO];
    if (![self load:(_menu.page + 1) * _menu.page_size] || (_menu.page + 1) * _menu.page_size >= _menu.count) return NO;
    return [self setGrid:YES];
}
// Expands with the current page as the first row, or collapses to the
// current page; the highlight stays on its candidate.
- (BOOL)setGrid:(BOOL)expanded {
    if (!arion_menu_grid(&_menu, self.engine.api, self.session, expanded)) return NO;
    return [self publishMenu];
}
- (BOOL)moveInRow:(NSInteger)delta {
    if (![self verify] || !arion_menu_move(&_menu, (int)delta)) return NO;
    return [self publishMenu];
}
// Loads engine candidates until index exists or the engine runs out.
- (BOOL)load:(NSUInteger)index {
    return arion_menu_load(&_menu, self.engine.api, self.session, index);
}
- (BOOL)turn:(NSInteger)delta {
    if (![self verify] || !arion_menu_turn(&_menu, self.engine.api, self.session, (int)delta)) return NO;
    return [self publishMenu];
}
// Menu-only operations still surface engine or learning storage failures.
- (BOOL)verify {
    ArionKeyboardSnapshot *snapshot = [self.engine snapshot:self.session error:nil];
    if (snapshot && !snapshot.failure) return YES;
    self.recoveryText = self.snapshot.preedit ?: self.recoveryText;
    self.available = NO;
    [self cancel];
    return NO;
}
- (NSString *)context {
    if (!self.previousCommit || NSDate.timeIntervalSinceReferenceDate - self.previousCommitTime > kContextLifetime) return nil;
    return self.previousCommit;
}
// Rebuilds the menu for a new engine state: the ranked window, page one.
- (BOOL)buildMenu {
    ArionContextRanker *ranker = self.contextRanking ? self.engine.ranker : nil;
    NSDictionary *ranking = ranker ? @{ @"ranker": ranker, @"previous": self.context ?: @"" } : nil;
    return arion_menu_build_typed(&_menu, self.engine.api, self.session, self.pageSize,
        ranking ? rankSharedMenu : NULL, (__bridge void *)ranking, sharedMenuMetadata, NULL);
}
// Shows the current page and makes the engine's highlight match Arion's, so
// a key the engine resolves itself (punctuation) commits the same candidate.
- (BOOL)publishMenu {
    [self freezeRanking];
    return [self publishRankedMenu];
}
- (BOOL)publishRankedMenu {
    ArionKeyboardSnapshot *base = self.engineSnapshot;
    if (!base) return NO;
    if (!arion_menu_prepare(&_menu, self.engine.api, self.session)) return NO;
    NSUInteger first = MIN(_menu.page * _menu.page_size, _menu.count);
    NSUInteger count = arion_menu_visible_count(&_menu);
    ArionKeyboardSnapshot *visible = [ArionKeyboardSnapshot new];
    visible.preedit = base.preedit;
    visible.selection = base.selection;
    visible.ascii = base.ascii;
    visible.commit = base.commit;
    visible.failure = base.failure;
    visible.candidates = sharedMenuTexts(&_menu, first, count);
    visible.comments = @[];
    visible.page = _menu.page;
    visible.lastPage = first + count >= _menu.count;
    visible.highlighted = _menu.highlight >= first ? _menu.highlight - first : 0;
    visible.expanded = _menu.expanded;
    if (_menu.expanded) {
        NSMutableArray *rows = [NSMutableArray array];
        for (NSUInteger row = _menu.top_row; row < _menu.top_row + arion_menu_row_count(&_menu); row++) {
            NSUInteger start = row * _menu.page_size;
            [rows addObject:sharedMenuTexts(&_menu, start, MIN(_menu.page_size, _menu.count - start))];
        }
        visible.rows = rows;
        visible.currentRow = _menu.page - _menu.top_row;
        visible.moreRows = (_menu.top_row + rows.count) * _menu.page_size < _menu.count;
    } else {
        visible.rows = @[visible.candidates];
        visible.currentRow = 0;
    }
    self.snapshot = visible;
    self.revision++;
    if (self.changed) self.changed();
    return YES;
}
- (BOOL)refresh:(NSError **)error {
    uintptr_t session = self.session;
    ArionKeyboardSnapshot *snapshot = [self.engine snapshot:session error:error];
    if (!snapshot) {
        self.recoveryText = self.snapshot.preedit ?: self.recoveryText;
        self.available = NO;
        [self cancel];
        return NO;
    }
    self.revision++;
    self.snapshot = snapshot;
    self.engineSnapshot = snapshot;
    self.ascii = snapshot.ascii;
    if (snapshot.commit.length > 0) {
        // A real commit result, never a candidate's display string.
        self.composing = NO;
        NSString *context = self.context;
        [self.client insertText:snapshot.commit replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        if (self.session != session || self.engineGeneration != self.engine.generation) return YES;
        // Raw Return commits Latin input; only converted Chinese is learned.
        if (self.contextRanking && !self.emojiInComposition && containsHan(snapshot.commit))
            [self.engine.ranker learn:snapshot.commit after:context];
        self.previousCommit = self.emojiInComposition ? nil : snapshot.commit;
        self.previousCommitTime = NSDate.timeIntervalSinceReferenceDate;
        self.emojiInComposition = NO;
    }
    if (snapshot.failure) {
        self.recoveryText = snapshot.preedit;
        self.available = NO;
        if (error) *error = snapshot.failure;
        [self cancel];
        return NO;
    }
    if (snapshot.preedit.length > 0) {
        self.composing = YES;
        NSAttributedString *marked = [[NSAttributedString alloc] initWithString:snapshot.preedit
            attributes:@{NSUnderlineStyleAttributeName: @(NSUnderlineStyleSingle)}];
        [self.client setMarkedText:marked selectionRange:snapshot.selection
                 replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    } else if (self.composing) {
        self.composing = NO;
        [self.client setMarkedText:@"" selectionRange:NSMakeRange(0, 0)
                 replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    }
    if (self.composing && snapshot.candidates.count > 0) {
        if (![self buildMenu] || ![self publishMenu]) {
            self.recoveryText = snapshot.preedit;
            self.available = NO;
            [self cancel];
            return NO;
        }
        return YES;
    }
    [self freezeRanking];
    arion_menu_dispose(&_menu);

    if (self.changed) self.changed();
    return YES;
}
- (void)setNeuralRanking:(BOOL)enabled {
    if (_neuralRanking == enabled) return;
    [self freezeRanking];
    _neuralRanking = enabled;
}
- (void)freezeRanking {
    self.rankingCandidates = nil; self.rankingSlots = nil;
    self.rankingRevision++;
    if (self.neuralRanking && self.rankingChanged) self.rankingChanged(nil, self.rankingRevision);
}
- (void)skipRanking:(NSString *)reason count:(NSUInteger)count {
    if (self.rankingSkipped) self.rankingSkipped(reason, count, ++self.rankingRevision);
}
- (void)requestRanking {
    if (!self.neuralRanking || !self.rankingChanged || !self.composing || self.ascii) return;
    if (_menu.page || _menu.highlight || _menu.expanded) { [self skipRanking:@"navigation" count:0]; return; }
    if (!_menu.count) { [self skipRanking:@"no_comparable_candidates" count:0]; return; }
    const char *input = self.engine.api->get_input(self.session);
    if (!input || !*input || strlen(input)>256 || [self.context lengthOfBytesUsingEncoding:NSUTF8StringEncoding]>2048) {
        [self skipRanking:@"input_bounds" count:0]; return;
    }
    if (self.engine.api->get_caret_pos(self.session)!=strlen(input)) { [self skipRanking:@"caret_not_at_end" count:0]; return; }
    if (!containsHan(@(_menu.texts[0]))) { [self skipRanking:@"non_chinese" count:0]; return; }
    NSMutableArray *candidates = [NSMutableArray array], *slots = [NSMutableArray array];
    for (size_t i=0; i<MIN(_menu.count,27); ++i) {
        ArionCandidateMetadata item = _menu.metadata[i];
        if (item.emoji || item.start != _menu.metadata[0].start || item.end != _menu.metadata[0].end ||
            item.end <= item.start || !containsHan(@(_menu.texts[i]))) continue;
        [slots addObject:@(i)];
        [candidates addObject:@{@"id":[NSString stringWithFormat:@"c%zu",_menu.indices[i]],
            @"text":@(_menu.texts[i]),@"source":@(item.type),@"start":@(item.start),@"end":@(item.end)}];
    }
    if (candidates.count==0) {
        [self skipRanking:@"no_comparable_candidates" count:candidates.count];
        return;
    }
    self.rankingCandidates = candidates; self.rankingSlots = slots;
    // Freeze and the new proposal are distinct messages on the bridge.
    self.rankingRevision++;
    self.rankingChanged(@{@"allowCompletion":(_menu.metadata[0].start==0 && _menu.metadata[0].end==strlen(input) ? @YES : @NO),@"input":@(input),@"preedit":self.snapshot.preedit ?: @"",@"before":self.context ?: @"",@"candidates":candidates},self.rankingRevision);
}
- (void)applyRankingMessage:(NSDictionary *)message activation:(NSString *)activation {
    uint64_t revision=0, generation=0, resultRevision=0;
    if (!self.live || !self.composing || !self.neuralRanking || !self.rankingCandidates || ![message[@"activation"] isEqual:activation] ||
        !vocabularyInteger(message[@"revision"],&revision) || revision!=self.rankingRevision) return;
    NSDictionary *result=message[@"rankResult"];
    if (![message[@"status"] isEqual:@"ok"] || ![result isKindOfClass:NSDictionary.class] ||
        ![result[@"session_id"] isEqual:activation] || !vocabularyInteger(result[@"generation"],&generation) || generation!=1 ||
        !vocabularyInteger(result[@"revision"],&resultRevision) || resultRevision!=revision) { return; }
    NSDictionary *completion=result[@"completion"];
    if (completion) {
        if (![completion isKindOfClass:NSDictionary.class] || ![completion[@"input"] isKindOfClass:NSString.class] ||
            ![completion[@"text"] isKindOfClass:NSString.class] || ![self verify]) return;
        if (!arion_completion_insert(self.session,[completion[@"input"] UTF8String],[completion[@"text"] UTF8String])) return;
        // Rebuild from real engine candidates. Do not allow the old pair ranker
        // to supersede the contextual completion before it is displayed.
        if (!arion_menu_build_typed(&_menu,self.engine.api,self.session,self.pageSize,NULL,NULL,sharedMenuMetadata,NULL) || ![self publishRankedMenu]) {
            self.recoveryText=self.snapshot.preedit; self.available=NO; [self cancel]; return;
        }
        [self freezeRanking];
        return;
    }
    NSArray *order=result[@"candidates"];
    if (![order isKindOfClass:NSArray.class] || order.count!=self.rankingCandidates.count) return;
    for (id item in order) if (![item isKindOfClass:NSDictionary.class] || ![item[@"id"] isKindOfClass:NSString.class]) return;
    NSUInteger chosen=NSNotFound;
    for (NSUInteger i=0;i<self.rankingCandidates.count;++i)
        if ([self.rankingCandidates[i][@"id"] isEqual:order[0][@"id"]]) chosen=i;
    if (chosen==NSNotFound) return;
    NSMutableArray *expected=[self.rankingCandidates mutableCopy];
    NSDictionary *selected=expected[chosen]; [expected removeObjectAtIndex:chosen]; [expected insertObject:selected atIndex:0];
    for (NSUInteger i=0;i<order.count;++i) if (![order[i][@"id"] isEqual:expected[i][@"id"]]) return;
    size_t slots[27]; for (NSUInteger i=0;i<self.rankingSlots.count;++i) slots[i]=self.rankingSlots[i].unsignedIntegerValue;
    if (![self verify] || ![self buildMenu] || !arion_menu_promote_slots(&_menu,slots,self.rankingSlots.count,chosen)) return;
    [self publishRankedMenu];
}
@end
