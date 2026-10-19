#import <Cocoa/Cocoa.h>
#import "keyboard.h"

// Owns keyboard composition independently of the asynchronous voice draft.
// The candidate menu is Arion's: the engine supplies candidates in its own
// order, the context ranker may reorder the leading ones, and Arion pages and
// highlights them. snapshot.candidates/page/lastPage/highlighted describe
// that menu's visible page.
@interface ArionTyping : NSObject
@property(readonly) BOOL composing;
@property(readonly) BOOL ascii;
@property(readonly) BOOL available;
@property(readonly, copy) NSString *recoveryText;
@property(readonly) NSUInteger revision;
@property(readonly, strong) ArionKeyboardSnapshot *snapshot;
@property(copy) void (^changed)(void);
// Candidates per page, 1–9. Applies from the next composition update.
@property(nonatomic) NSUInteger pageSize;
// Context ranking and its learning (上下文调频).
@property BOOL contextRanking;
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine;
- (BOOL)activate:(id)client error:(NSError **)error;
- (void)deactivate;
- (void)cancel;
- (void)toggleASCII;
- (BOOL)handleEvent:(NSEvent *)event;
// index is relative to the visible page.
- (BOOL)select:(NSUInteger)index revision:(NSUInteger)revision;
- (BOOL)highlight:(NSUInteger)index revision:(NSUInteger)revision;
- (BOOL)nextPage:(NSUInteger)revision;
@end
