#import "typing.h"
#import <InputMethodKit/InputMethodKit.h>

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
@end

@implementation ArionTyping
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine {
    if ((self = [super init])) self.engine = engine;
    return self;
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
}
- (void)cancel {
    if (self.engineGeneration == self.engine.generation) [self.engine clear:self.session];
    BOOL clear = self.composing;
    self.composing = NO;
    self.snapshot = nil;
    self.revision++;
    if (clear) [self.client setMarkedText:@"" selectionRange:NSMakeRange(0, 0)
                       replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    if (self.changed) self.changed();
}
- (void)toggleASCII {
    if (!self.session || !self.available || self.engineGeneration != self.engine.generation) return;
    [self cancel];
    [self.engine setASCII:!self.ascii session:self.session];
    [self refresh:nil];
}
- (BOOL)handleEvent:(NSEvent *)event {
    if (!self.session || !self.available || self.engineGeneration != self.engine.generation || event.type != NSEventTypeKeyDown) return NO;
    NSEventModifierFlags flags = event.modifierFlags;
    if (event.keyCode == 49 && (flags & NSEventModifierFlagShift) &&
        !(flags & (NSEventModifierFlagCommand | NSEventModifierFlagControl | NSEventModifierFlagOption))) {
        [self toggleASCII];
        return YES;
    }
    if (flags & (NSEventModifierFlagCommand | NSEventModifierFlagControl | NSEventModifierFlagOption)) {
        [self cancel];
        return NO;
    }
    int key = ArionKeyboardKey(event);
    if (key < 0) { [self cancel]; return NO; }
    BOOL handled = [self.engine processKey:key modifiers:ArionKeyboardModifiers(event) session:self.session];
    if (![self refresh:nil]) {
        NSBeep();
        return YES;
    }
    return handled;
}
- (BOOL)select:(NSUInteger)index revision:(NSUInteger)revision {
    if (!self.session || !self.available || self.engineGeneration != self.engine.generation || revision != self.revision || index >= self.snapshot.candidates.count) return NO;
    if (![self.engine select:index session:self.session]) return NO;
    return [self refresh:nil];
}
- (BOOL)highlight:(NSUInteger)index revision:(NSUInteger)revision {
    if (!self.session || !self.available || self.engineGeneration != self.engine.generation || revision != self.revision || index >= self.snapshot.candidates.count) return NO;
    if (![self.engine highlight:index session:self.session]) return NO;
    return [self refresh:nil];
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
    self.ascii = snapshot.ascii;
    if (snapshot.commit.length > 0) {
        // A real commit result, never a candidate's display string.
        self.composing = NO;
        [self.client insertText:snapshot.commit replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        if (self.session != session || self.engineGeneration != self.engine.generation) return YES;
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
    if (self.changed) self.changed();
    return YES;
}
@end
