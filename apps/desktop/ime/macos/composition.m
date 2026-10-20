#import "composition.h"
#import <InputMethodKit/InputMethodKit.h>

// The same spring as the dictation pill (ui/entities/dictation/typewriter.go,
// after Doubao's TypewriterSpring): a backlog near balance types at interval;
// each composed character above or below it speeds the pace up or down by
// elastic, within the bounds.
static const double kBalance = 6;
static const NSTimeInterval kInterval = 0.045;
static const double kElastic = 0.25;
static const NSTimeInterval kMinInterval = 0.012;
static const NSTimeInterval kMaxInterval = 0.090;
static const NSTimeInterval kTick = 1.0 / 60;

@interface ArionComposition ()
@property(weak) id client;
@property(readwrite, copy) NSString *activation;
@property(readwrite, copy) NSString *session;
@property(readwrite) BOOL live;
@property(readwrite) BOOL composing;
@property BOOL committed;
@property unsigned long long revision;
// The draft being typed, as composed characters, and how many are shown.
@property(copy) NSArray<NSString *> *target;
@property double shown;
@property(strong) NSTimer *timer;
@property NSTimeInterval lastTick;
@end

@implementation ArionComposition
- (instancetype)init {
    if ((self = [super init])) self.pacing = YES;
    return self;
}

- (void)dealloc {
    [_timer invalidate];
}

- (void)stopTyping {
    [self.timer invalidate];
    self.timer = nil;
    self.target = nil;
    self.shown = 0;
}

static NSArray<NSString *> *characters(NSString *text) {
    NSMutableArray *result = [NSMutableArray array];
    [text enumerateSubstringsInRange:NSMakeRange(0, text.length) options:NSStringEnumerationByComposedCharacterSequences
        usingBlock:^(NSString *s, __unused NSRange r, __unused NSRange e, __unused BOOL *stop) { [result addObject:s]; }];
    return result;
}

- (void)showMarked:(NSString *)text {
    NSAttributedString *marked = [[NSAttributedString alloc] initWithString:text
        attributes:@{NSUnderlineStyleAttributeName: @(NSUnderlineStyleSingle)}];
    [self.client setMarkedText:marked selectionRange:NSMakeRange(text.length, 0)
              replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
}

// Types toward the latest draft. Characters already shown follow revisions
// in place, so the marked text never retreats; only new characters are
// typed. A draft shorter than what is shown cuts the display to it now.
- (void)typeToward:(NSString *)text {
    NSArray<NSString *> *target = characters(text);
    BOOL cut = (double)target.count < self.shown;
    self.shown = MIN(self.shown, (double)target.count);
    NSUInteger shown = (NSUInteger)self.shown;
    BOOL changed = !cut && shown > 0 && ![[self.target subarrayWithRange:NSMakeRange(0, MIN(shown, self.target.count))]
        isEqualToArray:[target subarrayWithRange:NSMakeRange(0, shown)]];
    self.target = target;
    if (cut || changed) [self showMarked:[[target subarrayWithRange:NSMakeRange(0, shown)] componentsJoinedByString:@""]];
    // The host may reenter and invalidate while the marked text is set.
    if (!self.live || !self.target) return;
    if (!self.timer) {
        self.lastTick = NSDate.timeIntervalSinceReferenceDate;
        __weak ArionComposition *weakSelf = self;
        self.timer = [NSTimer timerWithTimeInterval:kTick repeats:YES block:^(__unused NSTimer *timer) { [weakSelf tick]; }];
        [NSRunLoop.mainRunLoop addTimer:self.timer forMode:NSRunLoopCommonModes];
    }
    [self tick];
}

- (void)tick {
    if (!self.live || !self.target) { [self stopTyping]; return; }
    NSTimeInterval now = NSDate.timeIntervalSinceReferenceDate;
    NSTimeInterval elapsed = MIN(MAX(now - self.lastTick, 0), 0.25);
    self.lastTick = now;
    double backlog = (double)self.target.count - self.shown;
    double scale = MAX(0.1, 1 + kElastic * (backlog - kBalance));
    NSTimeInterval pace = MAX(kMinInterval, MIN(kMaxInterval, kInterval / scale));
    NSUInteger before = (NSUInteger)self.shown;
    self.shown = MIN((double)self.target.count, self.shown + elapsed / pace);
    NSUInteger count = (NSUInteger)self.shown;
    NSArray<NSString *> *target = self.target;
    if (count == target.count) { [self.timer invalidate]; self.timer = nil; }
    // The host may reenter and invalidate while the marked text is set.
    if (count != before || count == target.count) [self showMarked:[[target subarrayWithRange:NSMakeRange(0, count)] componentsJoinedByString:@""]];
}

- (void)activate:(id)client {
    [self invalidate];
    self.client = client;
    self.activation = NSUUID.UUID.UUIDString;
    self.session = nil;
}

- (void)invalidate {
    [self stopTyping];
    BOOL clear = self.composing;
    id client = self.client;
    self.composing = NO;
    self.live = NO;
    if (clear) {
        [client setMarkedText:@"" selectionRange:NSMakeRange(0, 0)
            replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
    }
}

- (NSDictionary *)acknowledgement {
    return @{@"op": @"committed", @"activation": self.activation, @"session": self.session};
}

- (NSDictionary *)apply:(NSDictionary *)message {
    if (!self.client || ![message[@"activation"] isEqual:self.activation]) return nil;
    NSString *op = message[@"op"];
    NSString *session = message[@"session"];
    if (![op isKindOfClass:NSString.class] || ![session isKindOfClass:NSString.class] || session.length == 0) return nil;
    if ([op isEqual:@"begin"]) {
        if ([session isEqual:self.session]) return nil;
        [self invalidate];
        self.session = session;
        self.revision = 0;
        self.committed = NO;
        self.live = YES;
        return nil;
    }
    if (![session isEqual:self.session]) return nil;
    if ([op isEqual:@"commit"] && self.committed) return [self acknowledgement];
    if (!self.live) return nil;
    if ([op isEqual:@"cancel"]) {
        [self invalidate];
        return nil;
    }
    NSString *text = message[@"text"];
    if (![text isKindOfClass:NSString.class]) return nil;
    if ([op isEqual:@"mark"]) {
        NSNumber *revision = message[@"revision"];
        if (![revision isKindOfClass:NSNumber.class] || revision.doubleValue < 1 ||
            revision.doubleValue != (double)revision.unsignedLongLongValue ||
            revision.unsignedLongLongValue <= self.revision) return nil;
        self.revision = revision.unsignedLongLongValue;
        self.composing = YES;
        // An empty draft releases the host's composition at once.
        if (self.pacing && text.length > 0) {
            [self typeToward:text];
        } else {
            [self stopTyping];
            [self showMarked:text];
        }
    } else if ([op isEqual:@"commit"]) {
        // Terminal before calling the host, which can reenter the input method.
        // The final text replaces whatever the typewriter had shown.
        [self stopTyping];
        self.live = NO;
        self.committed = YES;
        self.composing = NO;
        NSDictionary *ack = [self acknowledgement];
        [self.client insertText:text replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        return ack;
    }
    return nil;
}
@end
