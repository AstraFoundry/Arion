#import <Cocoa/Cocoa.h>
#import "keyboard.h"

// What an explicit switch changed, for its caret badge.
typedef NS_ENUM(NSInteger, ArionTypingSwitch) {
    ArionTypingSwitchLanguage,
    ArionTypingSwitchWidth,
    ArionTypingSwitchPunctuation,
};

// Owns keyboard composition independently of the asynchronous voice draft.
// The candidate menu is Arion's: the engine supplies candidates in its own
// order, the context ranker may reorder the leading ones, and Arion pages and
// highlights them. snapshot.candidates/page/lastPage/highlighted describe
// that menu's visible page.
@interface ArionTyping : NSObject
@property(readonly) BOOL composing;
@property(readonly) BOOL ascii;
// Chinese-mode options (KBD-07), shared by every client.
@property(readonly) BOOL fullWidth;
@property(readonly) BOOL englishPunctuation;
@property(readonly) BOOL available;
@property(readonly, copy) NSString *recoveryText;
@property(readonly) NSUInteger revision;
@property(readonly, strong) ArionKeyboardSnapshot *snapshot;
@property(copy) void (^changed)(void);
// Fires only after an explicit successful language, width or punctuation switch.
@property(copy) void (^modeChanged)(ArionTypingSwitch change);
// Candidates per page, 1–9. Applies from the next composition update.
@property(nonatomic) NSUInteger pageSize;
// Context ranking and its learning (上下文调频).
@property BOOL contextRanking;
@property BOOL emojiSuggestions;
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine;
- (BOOL)activate:(id)client error:(NSError **)error;
- (void)deactivate;
- (void)cancel;
- (void)toggleASCII;
// Both discard unconfirmed composition; NO when the engine is not live or the
// choice could not be saved.
- (BOOL)toggleFullWidth;
- (BOOL)toggleEnglishPunctuation;
- (BOOL)handleEvent:(NSEvent *)event;
// index is relative to the visible page.
- (BOOL)select:(NSUInteger)index revision:(NSUInteger)revision;
- (BOOL)highlight:(NSUInteger)index revision:(NSUInteger)revision;
- (BOOL)nextPage:(NSUInteger)revision;
// Chooses a candidate in a visible row of the expanded grid.
- (BOOL)select:(NSUInteger)index row:(NSUInteger)row revision:(NSUInteger)revision;
// The chevron: expands the bar into the grid, or collapses it.
- (BOOL)toggleExpanded:(NSUInteger)revision;
@end
