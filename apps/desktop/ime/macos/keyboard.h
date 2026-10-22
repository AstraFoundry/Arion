#import <Cocoa/Cocoa.h>
#import "ranking.h"

// Immutable output copied from a single synchronous engine revision.
@interface ArionKeyboardSnapshot : NSObject
@property(copy) NSString *preedit;
@property NSRange selection;
@property(copy) NSArray<NSString *> *candidates;
@property(copy) NSArray<NSString *> *comments;
@property NSUInteger highlighted;
@property NSUInteger page;
@property BOOL lastPage;
// The expanded grid (Arion's menu only): its visible rows, one page each,
// and which of them is the current page (candidates).
@property BOOL expanded;
@property(copy) NSArray<NSArray<NSString *> *> *rows;
@property NSUInteger currentRow;
@property BOOL moreRows;
@property BOOL ascii;
@property(copy) NSString *commit;
@property(strong) NSError *failure;
@end

// One process-wide Rime service; sessions never share client pointers.
@interface ArionKeyboardEngine : NSObject
@property(readonly) NSUInteger generation;
// Chinese-mode width and punctuation (KBD-07), shared by all sessions and
// persisted in the user directory.
@property(readonly) BOOL fullWidth;
@property(readonly) BOOL englishPunctuation;
// Nil when the encrypted context store could not open: candidates then keep
// the engine's order (RNK-03).
@property(readonly, strong) ArionContextRanker *ranker;
+ (instancetype)shared;
- (BOOL)startWithData:(NSString *)data userDirectory:(NSString *)user error:(NSError **)error;
- (uintptr_t)createSession;
- (void)stop;
- (BOOL)resetLearning:(NSError **)error;
- (void)setASCII:(BOOL)ascii session:(uintptr_t)session;
// NO when the choice could not be persisted; the option is then unchanged.
- (BOOL)setFullWidth:(BOOL)fullWidth session:(uintptr_t)session;
- (BOOL)setEnglishPunctuation:(BOOL)englishPunctuation session:(uintptr_t)session;
- (void)destroySession:(uintptr_t)session;
- (BOOL)processKey:(int)key modifiers:(int)modifiers session:(uintptr_t)session;
- (BOOL)select:(NSUInteger)index session:(uintptr_t)session;
- (BOOL)highlight:(NSUInteger)index session:(uintptr_t)session;
// The menu's candidate texts from start, at most limit, by engine index;
// nil for undecodable text.
- (NSArray<NSString *> *)candidates:(uintptr_t)session from:(NSUInteger)start limit:(NSUInteger)limit;
- (BOOL)selectCandidate:(NSUInteger)index session:(uintptr_t)session;
- (BOOL)highlightCandidate:(NSUInteger)index session:(uintptr_t)session;
- (void)clear:(uintptr_t)session;
- (ArionKeyboardSnapshot *)snapshot:(uintptr_t)session error:(NSError **)error;
@end

// Native virtual keys are mapped explicitly to the X11 key symbols Rime uses.
FOUNDATION_EXPORT int ArionKeyboardKey(NSEvent *event);
FOUNDATION_EXPORT int ArionKeyboardModifiers(NSEvent *event);
