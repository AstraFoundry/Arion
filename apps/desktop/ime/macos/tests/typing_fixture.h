#pragma once
#import <Cocoa/Cocoa.h>
#import "../typing.h"

// The client protocol adapter owns a real AppKit text store; conversion is real librime.
@interface TypingEditor : NSObject
@property(strong) NSTextView *view;
@property NSUInteger insertions;
@property NSUInteger markings;
@end
@implementation TypingEditor
- (instancetype)init {
    if ((self = [super init])) self.view = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 500, 200)];
    return self;
}
- (void)setMarkedText:(id)text selectionRange:(NSRange)selection replacementRange:(NSRange)replacement {
    self.markings++;
    [self.view setMarkedText:text selectedRange:selection replacementRange:replacement];
}
- (void)insertText:(id)text replacementRange:(NSRange)replacement {
    self.insertions++;
    [self.view insertText:text replacementRange:replacement];
}
@end

static inline void require(BOOL condition, NSString *description) {
    if (!condition) @throw [NSException exceptionWithName:@"AssertionFailure" reason:description userInfo:nil];
}
static inline NSEvent *key(NSString *characters, unsigned short code, NSEventModifierFlags flags) {
    return [NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:flags
        timestamp:0 windowNumber:0 context:nil characters:characters charactersIgnoringModifiers:characters
        isARepeat:NO keyCode:code];
}
static inline BOOL modifier(ArionTyping *typing, unsigned short code, NSEventModifierFlags flags, NSTimeInterval timestamp) {
    NSEvent *event = [NSEvent keyEventWithType:NSEventTypeFlagsChanged location:NSZeroPoint modifierFlags:flags
        timestamp:timestamp windowNumber:0 context:nil characters:@"" charactersIgnoringModifiers:@""
        isARepeat:NO keyCode:code];
    return [typing handleEvent:event];
}
static inline BOOL press(ArionTyping *typing, NSString *characters, unsigned short code) {
    return [typing handleEvent:key(characters, code, 0)];
}
static inline void type(ArionTyping *typing, NSString *input) {
    for (NSUInteger i = 0; i < input.length; i++) {
        NSString *character = [input substringWithRange:NSMakeRange(i, 1)];
        require(press(typing, character, 0), [@"engine handles synthetic letter " stringByAppendingString:character]);
    }
}
