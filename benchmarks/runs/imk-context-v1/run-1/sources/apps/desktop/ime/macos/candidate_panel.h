#import <Cocoa/Cocoa.h>

// Horizontal candidate bar drawn by the input method itself. Geometry and
// colors follow the Chatterfly level-2 preset documented in
// docs/research/chatterfly-candidate-ui.md. It never takes key focus; the
// controller keeps routing keys to the typing owner. All calls run on the
// main thread.
@interface ArionCandidatePanel : NSObject
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
