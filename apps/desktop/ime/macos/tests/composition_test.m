#import <Cocoa/Cocoa.h>
#import "../composition.h"

// A protocol adapter around a real AppKit text store, not a system IMK client.
@interface Editor : NSObject
@property(strong) NSTextView *view;
@property NSUInteger insertions;
@property(copy) void (^onInsert)(void);
@property(copy) void (^onMark)(void);
@end
@implementation Editor
- (instancetype)init {
    if ((self = [super init])) self.view = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 400, 200)];
    return self;
}
- (void)setMarkedText:(id)text selectionRange:(NSRange)selection replacementRange:(NSRange)replacement {
    [self.view setMarkedText:text selectedRange:selection replacementRange:replacement];
    if (self.onMark) self.onMark();
}
- (void)insertText:(id)text replacementRange:(NSRange)replacement {
    self.insertions++;
    [self.view insertText:text replacementRange:replacement];
    if (self.onInsert) self.onInsert();
}
@end

static void require(BOOL success, NSString *message) {
    if (!success) { NSLog(@"FAIL: %@", message); exit(1); }
}
static NSDictionary *message(ArionComposition *state, NSString *op, NSString *session, NSString *text, NSNumber *revision) {
    NSMutableDictionary *value = [@{@"activation": state.activation, @"op": op, @"session": session} mutableCopy];
    if (text) value[@"text"] = text;
    if (revision) value[@"revision"] = revision;
    return value;
}
static void begin(ArionComposition *state, NSString *session) {
    [state apply:message(state, @"begin", session, nil, nil)];
}
static void mark(ArionComposition *state, NSString *session, NSString *text, NSNumber *revision) {
    [state apply:message(state, @"mark", session, text, revision)];
}

int main(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        Editor *editor = [Editor new];
        ArionComposition *state = [ArionComposition new];
        // Ownership cases check each mark as it arrives; pacing is tested below.
        state.pacing = NO;
        [state activate:editor];
        editor.view.string = @"A😀Z";
        [editor.view setSelectedRange:NSMakeRange(3, 0)];
        begin(state, @"first");
        mark(state, @"first", @"张伟", @2);
        require([editor.view.string isEqual:@"A😀张伟Z"], @"marked text preserves adjacent surrogate pair");
        require(editor.view.markedRange.length == 2, @"marked range uses UTF-16");
        mark(state, @"first", @"STALE", @1);
        mark(state, @"foreign", @"FOREIGN", @3);
        require([editor.view.string isEqual:@"A😀张伟Z"], @"older revision and foreign session ignored");
        NSDictionary *commit = message(state, @"commit", @"first", @"张伟😀", nil);
        require([state apply:commit] != nil, @"commit acknowledged");
        [state apply:commit];
        mark(state, @"first", @"LATE", @4);
        require(editor.insertions == 1, @"duplicate commit inserts exactly once");
        require([editor.view.string isEqual:@"A😀张伟😀Z"], @"commit exact text and boundaries");
        require(!editor.view.hasMarkedText && !state.live, @"commit terminal with no marked range");
        begin(state, @"first");
        require(!state.live, @"duplicate begin cannot revive a completed session");

        begin(state, @"cancelled");
        mark(state, @"cancelled", @"草稿", @1);
        [state apply:message(state, @"cancel", @"first", nil, nil)];
        require(state.live, @"old cancel does not clear current draft");
        [state apply:message(state, @"cancel", @"cancelled", nil, nil)];
        [state apply:message(state, @"commit", @"cancelled", @"LATE", nil)];
        require([editor.view.string isEqual:@"A😀张伟😀Z"] && editor.insertions == 1, @"cancel removes only owned draft and rejects late commit");

        begin(state, @"interrupted");
        mark(state, @"interrupted", @"draft", @1);
        [state invalidate];
        [editor.view insertText:@"USER" replacementRange:NSMakeRange(NSNotFound, NSNotFound)];
        [state apply:message(state, @"commit", @"interrupted", @"LATE", nil)];
        require([editor.view.string isEqual:@"A😀张伟😀USERZ"], @"host edit wins after interruption");

        begin(state, @"old-client");
        NSDictionary *old = message(state, @"mark", @"old-client", @"OLD", @1);
        Editor *other = [Editor new];
        [state activate:other];
        begin(state, @"new-client");
        [state apply:old];
        require(other.view.string.length == 0, @"old activation never writes to new client");
        mark(state, @"new-client", @"😀", @1);
        require(other.view.markedRange.length == 2 && other.view.selectedRange.location == 2, @"emoji caret uses UTF-16");
        mark(state, @"new-client", @"", @2);
        [state invalidate];
        require(!other.view.hasMarkedText && other.view.string.length == 0, @"empty draft still releases host composition");

        begin(state, @"malformed");
        [state apply:@{@"activation": state.activation, @"op": @"mark", @"session": @"malformed", @"text": @42, @"revision": @1}];
        mark(state, @"malformed", @"INVALID", @(-1));
        mark(state, @"malformed", @"INVALID", @1.5);
        require(other.view.string.length == 0, @"invalid text and revisions cannot mutate host");
        // Host callbacks can synchronously deactivate or activate another client.
        Editor *reentrant = [Editor new];
        [state activate:reentrant];
        begin(state, @"reentrant-commit");
        NSString *oldActivation = state.activation;
        reentrant.onInsert = ^{ [state activate:editor]; };
        NSDictionary *ack = [state apply:message(state, @"commit", @"reentrant-commit", @"done", nil)];
        require([ack[@"activation"] isEqual:oldActivation] && [ack[@"session"] isEqual:@"reentrant-commit"],
                @"commit acknowledgement remains bound across host reentrancy");
        require(!state.live && !state.composing, @"reentrant activation keeps new state idle");
        [state activate:reentrant];
        begin(state, @"reentrant-mark");
        __weak Editor *weakReentrant = reentrant;
        reentrant.onMark = ^{ weakReentrant.onMark = nil; [state invalidate]; [state activate:other]; };
        mark(state, @"reentrant-mark", @"temporary", @1);
        require(!state.live && !state.composing && !reentrant.view.hasMarkedText,
                @"reentrant mark cancellation never resurrects ownership");
        // Paced drafts type into the marked text over time; revisions keep the
        // agreeing prefix; commit replaces the draft exactly and at once.
        Editor *paced = [Editor new];
        ArionComposition *typing = [ArionComposition new];
        [typing activate:paced];
        begin(typing, @"paced");
        // Count delivered host updates, rather than samples of a busy run loop.
        __block NSUInteger previous = paced.view.string.length, steps = 0;
        __weak Editor *weakPaced = paced;
        paced.onMark = ^{
            NSUInteger length = weakPaced.view.string.length;
            if (length != previous) { steps++; previous = length; }
        };
        mark(typing, @"paced", @"今天下午三点我们开会", @1);
        require(paced.view.string.length < 10, @"a paced draft does not appear whole");
        NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:2];
        while (paced.view.string.length < 10 && [deadline timeIntervalSinceNow] > 0) {
            [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:0.02]];
        }
        require([paced.view.string isEqual:@"今天下午三点我们开会"] && paced.view.hasMarkedText && steps >= 4,
                @"the draft types out in several steps");
        paced.onMark = nil;
        mark(typing, @"paced", @"今天下午三点他们开会", @2);
        require([paced.view.string isEqual:@"今天下午三点他们开会"], @"a revision changes shown characters in place");
        mark(typing, @"paced", @"今天下午 3 点他们开会", @3);
        require(paced.view.string.length >= 10, @"an early rewrite never retreats the marked text");
        mark(typing, @"paced", @"今天", @4);
        require([paced.view.string isEqual:@"今天"], @"a shorter draft cuts the display at once");
        mark(typing, @"paced", @"今天下午三点他们开会讨论新版本的发布计划", @5);
        [typing apply:message(typing, @"commit", @"paced", @"今天下午三点他们开会讨论新版本的发布计划。", nil)];
        require([paced.view.string isEqual:@"今天下午三点他们开会讨论新版本的发布计划。"] && !paced.view.hasMarkedText && paced.insertions == 1,
                @"commit inserts the final text exactly while typing is behind");
        [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:0.2]];
        require(!paced.view.hasMarkedText && paced.insertions == 1, @"no typing continues after commit");
        begin(typing, @"paced-cancel");
        mark(typing, @"paced-cancel", @"草稿草稿草稿", @1);
        [typing apply:message(typing, @"cancel", @"paced-cancel", @"", nil)];
        [[NSRunLoop currentRunLoop] runUntilDate:[NSDate dateWithTimeIntervalSinceNow:0.2]];
        require(!paced.view.hasMarkedText, @"no typing continues after cancel");
        NSLog(@"PASS: native AppKit composition ownership, terminality, revisions and Unicode");
    }
    return 0;
}
