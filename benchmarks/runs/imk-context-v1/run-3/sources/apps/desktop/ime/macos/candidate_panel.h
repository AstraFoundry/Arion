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

// Horizontal candidate bar drawn by the input method itself. Geometry and
// colors follow Chatterfly's shipped presets and themes. It never takes key
// focus; the controller keeps routing keys to the typing owner. The chrome,
// shadows and selection are Core Animation layers composited by the window
// server; only the text redraws per keystroke. All calls run on the main thread.
@interface ArionCandidatePanel : NSObject
@property(nonatomic) ArionCandidateSize size;
@property(nonatomic) ArionCandidateAppearance appearance;
// Index into the shown page; called for a click on a candidate.
@property(copy) void (^selected)(NSUInteger index, NSUInteger revision);
// Called for a click on the page chevron.
@property(copy) void (^nextPage)(NSUInteger revision);
- (void)showCandidates:(NSArray<NSString *> *)candidates
           highlighted:(NSUInteger)highlighted
              lastPage:(BOOL)lastPage
              revision:(NSUInteger)revision
             caretRect:(NSRect)caret;
- (void)hide;
@end
