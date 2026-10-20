#import <Cocoa/Cocoa.h>

// Chatterfly's first three size levels (docs/research/chatterfly-candidate-ui.md).
typedef NS_ENUM(NSInteger, ArionCandidateSize) {
    ArionCandidateSizeSmall,
    ArionCandidateSizeMedium,
    ArionCandidateSizeLarge,
};

typedef NS_ENUM(NSInteger, ArionCandidateAppearance) {
    ArionCandidateAppearanceSystem,
    ArionCandidateAppearanceLight,
    ArionCandidateAppearanceDark,
};

// Candidate bar drawn by the input method itself: one horizontal row, or the
// expanded grid of up to six rows (one page each, columns aligned), as
// Chatterfly's spread list. Geometry and colors follow Chatterfly's shipped
// presets and themes. It never takes key focus; the controller keeps routing
// keys to the typing owner. The chrome, shadows, row tint and selection are
// Core Animation layers composited by the window server; only the text
// redraws per keystroke. Expanding and collapsing animate those layers over
// 0.3 s and 0.2 s, Chatterfly's durations. All calls run on the main thread.
@interface ArionCandidatePanel : NSObject
@property(nonatomic) ArionCandidateSize size;
@property(nonatomic) ArionCandidateAppearance appearance;
// A click on a candidate: its index in its visible row, and that row.
@property(copy) void (^selected)(NSUInteger index, NSUInteger row, NSUInteger revision);
// A click on the chevron, which expands the bar or collapses the grid.
@property(copy) void (^toggled)(NSUInteger revision);
// rows[currentRow] is the current page; highlighted indexes it. more says a
// later page exists (the collapsed bar shows the expand chevron).
- (void)showRows:(NSArray<NSArray<NSString *> *> *)rows
      currentRow:(NSUInteger)currentRow
     highlighted:(NSUInteger)highlighted
        expanded:(BOOL)expanded
            more:(BOOL)more
        revision:(NSUInteger)revision
       caretRect:(NSRect)caret;
// The collapsed bar for one page.
- (void)showCandidates:(NSArray<NSString *> *)candidates
           highlighted:(NSUInteger)highlighted
              lastPage:(BOOL)lastPage
              revision:(NSUInteger)revision
             caretRect:(NSRect)caret;
- (void)hide;
@end

// Brief, click-through language feedback at the host caret.
@interface ArionLanguagePanel : NSObject
@property(nonatomic) ArionCandidateAppearance appearance;
- (void)showASCII:(BOOL)ascii caretRect:(NSRect)caret;
- (void)hide;
@end
