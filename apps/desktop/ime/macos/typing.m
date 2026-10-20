#import "typing.h"
#import <InputMethodKit/InputMethodKit.h>

// The engine's leading candidates the ranker may reorder; later pages keep
// engine order and load on demand.
static const NSUInteger kRankWindow = 27;
// Rows the expanded grid shows at once, as Chatterfly's expandedRowCount.
static const NSUInteger kExpandedRows = 6;
// A previous commit older than this no longer predicts the next word.
static const NSTimeInterval kContextLifetime = 120;

static BOOL containsHan(NSString *text) {
    static NSRegularExpression *han;
    static dispatch_once_t once;
    dispatch_once(&once, ^{ han = [NSRegularExpression regularExpressionWithPattern:@"\\p{Han}" options:0 error:nil]; });
    return [han firstMatchInString:text options:0 range:NSMakeRange(0, text.length)] != nil;
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
// Menu in display order, with each entry's engine index.
@property(strong) NSMutableArray<NSString *> *menuTexts;
@property(strong) NSMutableArray<NSNumber *> *menuIndices;
@property NSUInteger fetched;
@property BOOL exhausted;
@property NSUInteger page;
@property NSUInteger highlightIndex;
@property BOOL expanded;
// First page shown as a row of the expanded grid.
@property NSUInteger topRow;
@property(copy) NSString *previousCommit;
@property NSTimeInterval previousCommitTime;
@end

@implementation ArionTyping
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine {
    if ((self = [super init])) {
        self.engine = engine;
        self.pageSize = 9;
        self.contextRanking = YES;
    }
    return self;
}
- (void)setPageSize:(NSUInteger)pageSize {
    _pageSize = MAX(1, MIN(9, pageSize));
}
- (void)dealloc {
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
    if (self.engineGeneration == self.engine.generation) [self.engine clear:self.session];
    BOOL clear = self.composing;
    self.composing = NO;
    self.expanded = NO;
    self.topRow = 0;
    self.snapshot = nil;
    self.engineSnapshot = nil;
    self.menuTexts = nil;
    self.menuIndices = nil;
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
    [self refresh:nil];
}
- (BOOL)live {
    return self.session && self.available && self.engineGeneration == self.engine.generation;
}
- (BOOL)handleEvent:(NSEvent *)event {
    if (!self.live || event.type != NSEventTypeKeyDown) return NO;
    NSEventModifierFlags flags = event.modifierFlags;
    if (event.keyCode == 49 && (flags & NSEventModifierFlagShift) &&
        !(flags & (NSEventModifierFlagCommand | NSEventModifierFlagControl | NSEventModifierFlagOption))) {
        [self toggleASCII];
        return YES;
    }
    if (flags & (NSEventModifierFlagCommand | NSEventModifierFlagControl | NSEventModifierFlagOption)) {
        [self cancel];
        self.previousCommit = nil;
        return NO;
    }
    if (self.composing && self.menuTexts.count > 0 && !self.ascii) {
        NSInteger handled = [self menuKey:event];
        if (handled >= 0) return handled;
    }
    int key = ArionKeyboardKey(event);
    if (key < 0) { [self cancel]; self.previousCommit = nil; return NO; }
    BOOL wasComposing = self.composing;
    BOOL handled = [self.engine processKey:key modifiers:ArionKeyboardModifiers(event) session:self.session];
    if (![self refresh:nil]) {
        NSBeep();
        return YES;
    }
    // A key the host handles (Return, arrows, Delete) may move its caret away
    // from the last commit, which then stops being context.
    if (!handled && !wasComposing) self.previousCommit = nil;
    return handled;
}
// Keys that act on Arion's menu. Returns -1 for keys the engine handles.
- (NSInteger)menuKey:(NSEvent *)event {
    NSString *characters = event.characters;
    unichar character = characters.length == 1 ? [characters characterAtIndex:0] : 0;
    NSUInteger first = self.page * self.pageSize;
    if (character >= '1' && character <= '9') {
        NSUInteger index = first + (NSUInteger)(character - '1');
        // A number past the page's end selects nothing but stays a selection key.
        if (character - '1' < (int)self.pageSize && index < self.menuTexts.count) [self choose:index];
        return YES;
    }
    if (character == ' ') { [self choose:self.highlightIndex]; return YES; }
    BOOL previous = character == '-' || character == '[' || event.keyCode == 116 || event.keyCode == 126;
    BOOL next = character == '=' || character == ']' || event.keyCode == 121 || event.keyCode == 125;
    if (self.expanded) {
        // Rows are pages: move between them; above the first row collapses.
        if (previous) { if (self.page == 0) [self setGrid:NO]; else [self turn:-1]; return YES; }
        if (next) { [self turn:1]; return YES; }
        if (event.keyCode == 123 || event.keyCode == 124) { [self moveInRow:event.keyCode == 123 ? -1 : 1]; return YES; }
        return -1;
    }
    if (previous) { [self turn:-1]; return YES; }
    // The next page opens the grid with that page as the current row, as
    // Chatterfly's spread list; a single page has nothing to expand.
    if (next) {
        if ([self verify] && [self load:(self.page + 1) * self.pageSize] && (self.page + 1) * self.pageSize < self.menuTexts.count) {
            self.expanded = YES;
            self.topRow = self.page;
            if (![self turn:1]) self.expanded = NO;
        }
        return YES;
    }
    return -1;
}
- (BOOL)choose:(NSUInteger)index {
    if (index >= self.menuIndices.count) return NO;
    if (![self.engine selectCandidate:self.menuIndices[index].unsignedIntegerValue session:self.session]) return NO;
    return [self refresh:nil];
}
- (BOOL)select:(NSUInteger)index revision:(NSUInteger)revision {
    if (!self.live || revision != self.revision || index >= self.snapshot.candidates.count) return NO;
    return [self choose:self.page * self.pageSize + index];
}
- (BOOL)highlight:(NSUInteger)index revision:(NSUInteger)revision {
    if (!self.live || revision != self.revision || index >= self.snapshot.candidates.count || ![self verify]) return NO;
    self.highlightIndex = self.page * self.pageSize + index;
    return [self publishMenu];
}
- (BOOL)nextPage:(NSUInteger)revision {
    if (!self.live || revision != self.revision || !self.composing) return NO;
    return [self turn:1];
}
- (BOOL)select:(NSUInteger)index row:(NSUInteger)row revision:(NSUInteger)revision {
    if (!self.live || revision != self.revision || row >= self.snapshot.rows.count || index >= self.snapshot.rows[row].count) return NO;
    return [self choose:(self.topRow + row) * self.pageSize + index];
}
- (BOOL)toggleExpanded:(NSUInteger)revision {
    if (!self.live || revision != self.revision || !self.composing || ![self verify]) return NO;
    if (self.expanded) return [self setGrid:NO];
    if (![self load:(self.page + 1) * self.pageSize] || (self.page + 1) * self.pageSize >= self.menuTexts.count) return NO;
    return [self setGrid:YES];
}
// Expands with the current page as the first row, or collapses to the
// current page; the highlight stays on its candidate.
- (BOOL)setGrid:(BOOL)expanded {
    self.expanded = expanded;
    self.topRow = self.page;
    return [self publishMenu];
}
- (BOOL)moveInRow:(NSInteger)delta {
    NSUInteger first = self.page * self.pageSize;
    NSInteger index = (NSInteger)self.highlightIndex + delta;
    if (index < (NSInteger)first || index >= (NSInteger)MIN(first + self.pageSize, self.menuTexts.count) || ![self verify]) return NO;
    self.highlightIndex = (NSUInteger)index;
    return [self publishMenu];
}
// Loads engine candidates until index exists or the engine runs out.
- (BOOL)load:(NSUInteger)index {
    while (index >= self.menuTexts.count && !self.exhausted) {
        NSUInteger limit = MAX(self.pageSize * 3, index + 1 - self.menuTexts.count);
        NSArray<NSString *> *more = [self.engine candidates:self.session from:self.fetched limit:limit];
        if (!more) return NO;
        for (NSUInteger i = 0; i < more.count; i++) {
            [self.menuTexts addObject:more[i]];
            [self.menuIndices addObject:@(self.fetched + i)];
        }
        self.fetched += more.count;
        self.exhausted = more.count < limit;
    }
    return YES;
}
- (BOOL)turn:(NSInteger)delta {
    if ((delta < 0 && self.page == 0) || ![self verify]) return NO;
    NSUInteger page = (NSUInteger)((NSInteger)self.page + delta);
    if (![self load:page * self.pageSize] || page * self.pageSize >= self.menuTexts.count) return NO;
    self.page = page;
    self.highlightIndex = page * self.pageSize;
    // The grid scrolls by whole rows to keep the current row visible.
    if (page < self.topRow) self.topRow = page;
    if (page >= self.topRow + kExpandedRows) self.topRow = page + 1 - kExpandedRows;
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
    self.menuTexts = [NSMutableArray array];
    self.menuIndices = [NSMutableArray array];
    self.fetched = 0;
    self.exhausted = NO;
    self.page = 0;
    self.highlightIndex = 0;
    // Typing changes the candidates; the grid closes with the old ones.
    self.expanded = NO;
    self.topRow = 0;
    NSArray<NSString *> *window = [self.engine candidates:self.session from:0 limit:MAX(kRankWindow, self.pageSize + 1)];
    if (!window) return NO;
    self.fetched = window.count;
    self.exhausted = window.count < MAX(kRankWindow, self.pageSize + 1);
    ArionContextRanker *ranker = self.contextRanking ? self.engine.ranker : nil;
    NSArray<NSNumber *> *order = ranker ? [ranker order:window after:self.context] : nil;
    for (NSUInteger i = 0; i < window.count; i++) {
        NSUInteger source = order ? order[i].unsignedIntegerValue : i;
        [self.menuTexts addObject:window[source]];
        [self.menuIndices addObject:@(source)];
    }
    return YES;
}
// Shows the current page and makes the engine's highlight match Arion's, so
// a key the engine resolves itself (punctuation) commits the same candidate.
- (BOOL)publishMenu {
    ArionKeyboardSnapshot *base = self.engineSnapshot;
    if (!base) return NO;
    // The engine reports false when the highlight is already there.
    if (self.highlightIndex < self.menuIndices.count)
        [self.engine highlightCandidate:self.menuIndices[self.highlightIndex].unsignedIntegerValue session:self.session];
    NSUInteger first = MIN(self.page * self.pageSize, self.menuTexts.count);
    NSUInteger count = MIN(self.pageSize, self.menuTexts.count - first);
    // The page is last once nothing follows it, loading one more if needed.
    if (![self load:first + count]) return NO;
    ArionKeyboardSnapshot *visible = [ArionKeyboardSnapshot new];
    visible.preedit = base.preedit;
    visible.selection = base.selection;
    visible.ascii = base.ascii;
    visible.commit = base.commit;
    visible.failure = base.failure;
    visible.candidates = [self.menuTexts subarrayWithRange:NSMakeRange(first, count)];
    visible.comments = @[];
    visible.page = self.page;
    visible.lastPage = first + count >= self.menuTexts.count;
    visible.highlighted = self.highlightIndex >= first ? self.highlightIndex - first : 0;
    visible.expanded = self.expanded;
    if (self.expanded) {
        NSMutableArray *rows = [NSMutableArray array];
        if (![self load:(self.topRow + kExpandedRows) * self.pageSize]) return NO;
        for (NSUInteger row = self.topRow; row < self.topRow + kExpandedRows && row * self.pageSize < self.menuTexts.count; row++) {
            NSUInteger start = row * self.pageSize;
            [rows addObject:[self.menuTexts subarrayWithRange:NSMakeRange(start, MIN(self.pageSize, self.menuTexts.count - start))]];
        }
        visible.rows = rows;
        visible.currentRow = self.page - self.topRow;
        visible.moreRows = (self.topRow + rows.count) * self.pageSize < self.menuTexts.count;
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
        if (self.contextRanking && containsHan(snapshot.commit)) [self.engine.ranker learn:snapshot.commit after:context];
        self.previousCommit = snapshot.commit;
        self.previousCommitTime = NSDate.timeIntervalSinceReferenceDate;
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
    self.menuTexts = nil;
    self.menuIndices = nil;
    if (self.changed) self.changed();
    return YES;
}
@end
