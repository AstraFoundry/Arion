// Replays a simulated user's typing stream through the production typing owner,
// candidate bar and an AppKit text view. The simulated user knows the word it
// wants, finds it in Arion's menu (paging as needed) and confirms it with the
// key a person would press. Ranking conditions differ only by contextRanking.
#import <Cocoa/Cocoa.h>
#import "typing.h"
#import "learning.h"
#import "candidate_panel.h"
#include <mach/mach_time.h>

static mach_timebase_info_data_t timebase;
static double since(uint64_t start) {
    return (double)(mach_absolute_time() - start) * timebase.numer / timebase.denom / 1000000.0;
}
static void require(BOOL condition, NSString *message) {
    if (!condition) @throw [NSException exceptionWithName:@"DriverFailure" reason:message userInfo:nil];
}

@interface Editor : NSObject
@property(strong) NSTextView *view;
@end
@implementation Editor
- (instancetype)init {
    if ((self = [super init])) self.view = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 640, 200)];
    return self;
}
- (void)setMarkedText:(id)text selectionRange:(NSRange)selection replacementRange:(NSRange)replacement {
    [self.view setMarkedText:text selectedRange:selection replacementRange:replacement];
}
- (void)insertText:(id)text replacementRange:(NSRange)replacement {
    [self.view insertText:text replacementRange:replacement];
}
- (NSDictionary *)attributesForCharacterIndex:(NSUInteger)index lineHeightRectangle:(NSRect *)rect {
    if (rect) *rect = NSMakeRect(200, 600, 1, 18);
    return @{};
}
@end

static const unsigned short digitCodes[] = {18, 19, 20, 21, 23, 22, 26, 28, 25};

int main(int argc, const char *argv[]) {
    @autoreleasepool {
        if (argc != 5) { fprintf(stderr, "usage: driver DATA USER REQUEST_JSON OUTPUT_JSON\n"); return 2; }
        mach_timebase_info(&timebase);
        [NSApplication sharedApplication];
        NSString *data = @(argv[1]), *user = @(argv[2]);
        NSMutableDictionary *result = [@{@"status": @"failed", @"timer": @"mach_absolute_time",
            @"boundary": @"ArionTyping.handleEvent (engine, ranking, menu, NSTextView marked/insert text) and, separately, candidate bar layout plus offscreen-backed draw; no OS event dispatch or compositor"} mutableCopy];
        int status = 2;
        ArionTyping *typing = nil;
        @try {
            NSDictionary *request = [NSJSONSerialization JSONObjectWithData:[NSData dataWithContentsOfFile:@(argv[3])] options:0 error:nil];
            require([request isKindOfClass:NSDictionary.class], @"read request");
            BOOL ranking = [request[@"contextRanking"] boolValue];
            NSUInteger pageSize = [request[@"pageSize"] unsignedIntegerValue];
            NSMutableData *key = [NSMutableData dataWithLength:32];
            memset(key.mutableBytes, 0x5a, key.length);
            NSError *error = nil;
            require(ArionRegisterTestEncryptedLearning([user stringByAppendingPathComponent:@"learning"], key, &error), @"register synthetic-key vault");
            ArionKeyboardEngine *engine = ArionKeyboardEngine.shared;
            require([engine startWithData:data userDirectory:user error:&error], @"start pinned engine");
            require(engine.ranker != nil, @"context store opens");
            Editor *editor = [Editor new];
            typing = [[ArionTyping alloc] initWithEngine:engine];
            typing.pageSize = pageSize;
            typing.contextRanking = ranking;
            require([typing activate:editor error:&error], @"activate typing session");
            if (typing.ascii) [typing toggleASCII];

            ArionCandidatePanel *panel = [ArionCandidatePanel new];
            NSWindow *window = [panel valueForKey:@"window"];
            NSMutableArray *panelMs = [NSMutableArray array];
            // Each menu change lays out and draws the bar, as the controller does.
            typing.changed = ^{
                uint64_t start = mach_absolute_time();
                ArionKeyboardSnapshot *s = typing.snapshot;
                if (typing.composing && s.candidates.count > 0) {
                    [panel showCandidates:s.candidates highlighted:s.highlighted lastPage:s.lastPage revision:typing.revision caretRect:NSMakeRect(200, 600, 1, 18)];
                    [window displayIfNeeded];
                } else {
                    [panel hide];
                }
                [panelMs addObject:@(since(start))];
            };

            NSMutableArray *letterMs = [NSMutableArray array], *selectMs = [NSMutableArray array], *steps = [NSMutableArray array];
            BOOL (^press)(NSString *, unsigned short, NSMutableArray *) = ^BOOL(NSString *characters, unsigned short code, NSMutableArray *times) {
                NSEvent *event = [NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:0 timestamp:0
                    windowNumber:0 context:nil characters:characters charactersIgnoringModifiers:characters isARepeat:NO keyCode:code];
                uint64_t start = mach_absolute_time();
                BOOL handled = [typing handleEvent:event];
                [times addObject:@(since(start))];
                return handled;
            };
            for (NSDictionary *step in request[@"steps"]) {
                NSString *pinyin = step[@"pinyin"], *word = step[@"word"];
                NSUInteger before = editor.view.string.length;
                for (NSUInteger i = 0; i < pinyin.length; i++) press([pinyin substringWithRange:NSMakeRange(i, 1)], 0, letterMs);
                require(typing.composing, @"composition starts");
                NSInteger rank = -1;
                NSUInteger keys = 0;
                for (NSUInteger page = 0; page < 6 && rank < 0; page++) {
                    NSUInteger index = [typing.snapshot.candidates indexOfObject:word];
                    if (index != NSNotFound) {
                        rank = (NSInteger)(page * pageSize + index);
                        if (index == typing.snapshot.highlighted) press(@" ", 49, selectMs);
                        else press([NSString stringWithFormat:@"%lu", (unsigned long)index + 1], digitCodes[index], selectMs);
                        keys++;
                        break;
                    }
                    if (typing.snapshot.lastPage) break;
                    press(@"", 121, selectMs);
                    keys++;
                }
                if (rank < 0) { press(@"\x1b", 53, selectMs); keys++; }
                // A prefix choice leaves the rest of the input composing; that is a miss.
                NSString *inserted = [editor.view.string substringFromIndex:MIN(before, editor.view.string.length)];
                BOOL committed = rank >= 0 && !typing.composing && [inserted isEqual:word];
                if (rank >= 0 && !committed) { [typing cancel]; }
                [steps addObject:@{@"index": step[@"index"], @"kind": step[@"kind"], @"pair": step[@"pair"], @"occurrence": step[@"occurrence"],
                    @"rank": @(rank), @"selectionKeys": @(keys), @"committed": @(committed)}];
            }
            [typing deactivate];
            [engine stop];
            result[@"status"] = @"completed";
            result[@"contextRanking"] = @(ranking);
            result[@"steps"] = steps;
            result[@"letterKeyMs"] = letterMs;
            result[@"selectionKeyMs"] = selectMs;
            result[@"candidateBarMs"] = panelMs;
            status = 0;
        } @catch (NSException *exception) {
            result[@"failure"] = exception.reason ?: exception.name;
        }
        NSData *json = [NSJSONSerialization dataWithJSONObject:result options:NSJSONWritingSortedKeys error:nil];
        [json writeToFile:@(argv[4]) atomically:YES];
        return status;
    }
}
