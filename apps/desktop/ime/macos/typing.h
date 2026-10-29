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
// Context ranking and its learning (上下文调频). Always on in the input
// method; tests turn it off to read the engine's own order.
@property BOOL contextRanking;
@property(nonatomic) BOOL neuralRanking;
// Nil snapshot freezes inference; callbacks run on the native owner thread.
@property(copy) void (^rankingChanged)(NSDictionary *snapshot, NSUInteger revision);
- (void)freezeRanking;
- (void)applyRankingMessage:(NSDictionary *)message activation:(NSString *)activation;
@property BOOL emojiSuggestions;
@property(nonatomic) BOOL typoCorrection;
@property(nonatomic) uint32_t fuzzyPinyin;
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine;
- (BOOL)activate:(id)client error:(NSError **)error;
- (void)deactivate;
- (void)cancel;
// Invoke only from a verified bridge on the engine/UI thread. A blocked voice
// session refuses publication. Unknown or stale-activation messages return nil.
- (NSDictionary *)applyVocabularyMessage:(NSDictionary *)message activation:(NSString *)activation blocked:(BOOL)blocked;
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
