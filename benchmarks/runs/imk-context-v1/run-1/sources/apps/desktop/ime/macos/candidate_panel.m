#import "candidate_panel.h"

// Chatterfly (Yuanbao profile) size level 2 and its light/dark themes, read
// from +[YuanbaoProductProfile yuanbaoCandidateSizePresets] and the
// candidate themes. See docs/research/chatterfly-candidate-ui.md.
static const CGFloat kBarHeight = 32;
static const CGFloat kOuterRadius = 10;
static const CGFloat kBorderWidth = 0.5;
static const CGFloat kSelectionHeight = 24;
static const CGFloat kSelectionRadius = 6;
static const CGFloat kSelectionPadding = 4;
static const CGFloat kContentInset = 4;
static const CGFloat kCandidateSpacing = 23;
static const CGFloat kBodyFontSize = 16;
static const CGFloat kNumberFontSize = 13;
static const CGFloat kNumberWidth = 8;
static const CGFloat kNumberTextSpacing = 3;
static const CGFloat kArrowSize = 20;
static const CGFloat kArrowTrailingInset = 8;
// Room for the drawn shadows (radius 8, offset 4) outside the bar.
static const CGFloat kShadowMargin = 14;
// Distance between the caret line and the bar.
static const CGFloat kCaretGap = 4;

static NSColor *srgb(CGFloat r, CGFloat g, CGFloat b, CGFloat a) {
    return [NSColor colorWithSRGBRed:r green:g blue:b alpha:a];
}

@interface ArionCandidateView : NSView
@property(copy) NSArray<NSString *> *candidates;
@property NSUInteger highlighted;
@property BOOL paged;
@property(strong) NSMutableArray<NSValue *> *itemRects;
@property NSRect arrowRect;
@property(copy) void (^clicked)(NSInteger index);
@property(copy) void (^clickedArrow)(void);
- (NSSize)layoutBar;
@end

@implementation ArionCandidateView
- (BOOL)isFlipped { return YES; }
- (BOOL)acceptsFirstMouse:(NSEvent *)event { return YES; }

- (BOOL)dark {
    return [[self.effectiveAppearance bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]]
        isEqualToString:NSAppearanceNameDarkAqua];
}

- (NSFont *)bodyFont {
    return [NSFont fontWithName:@"PingFangSC-Regular" size:kBodyFontSize] ?: [NSFont systemFontOfSize:kBodyFontSize];
}

- (NSFont *)numberFont {
    return [NSFont monospacedDigitSystemFontOfSize:kNumberFontSize weight:NSFontWeightRegular];
}

// Lays out every candidate as [number, gap, text] inside a selection-sized
// box and returns the bar size, excluding the shadow margin.
- (NSSize)layoutBar {
    self.itemRects = [NSMutableArray array];
    NSDictionary *body = @{NSFontAttributeName: self.bodyFont};
    CGFloat x = kShadowMargin + kContentInset;
    CGFloat top = kShadowMargin + (kBarHeight - kSelectionHeight) / 2;
    for (NSString *candidate in self.candidates) {
        CGFloat text = ceil([candidate sizeWithAttributes:body].width);
        CGFloat width = kSelectionPadding + kNumberWidth + kNumberTextSpacing + text + kSelectionPadding;
        [self.itemRects addObject:[NSValue valueWithRect:NSMakeRect(x, top, width, kSelectionHeight)]];
        x += width + kCandidateSpacing - 2 * kSelectionPadding;
    }
    x -= kCandidateSpacing - 2 * kSelectionPadding;
    if (self.paged) {
        x += kArrowTrailingInset;
        self.arrowRect = NSMakeRect(x, kShadowMargin + (kBarHeight - kArrowSize) / 2, kArrowSize, kArrowSize);
        x += kArrowSize + kArrowTrailingInset;
    } else {
        self.arrowRect = NSZeroRect;
        x += kContentInset;
    }
    return NSMakeSize(x - kShadowMargin, kBarHeight);
}

- (void)drawRect:(NSRect)dirty {
    BOOL dark = self.dark;
    NSRect bar = NSInsetRect(self.bounds, kShadowMargin, kShadowMargin);
    NSBezierPath *outline = [NSBezierPath bezierPathWithRoundedRect:bar xRadius:kOuterRadius yRadius:kOuterRadius];
    NSColor *background = dark ? srgb(23 / 255.0, 23 / 255.0, 23 / 255.0, 1) : srgb(1, 1, 1, 1);

    // Two soft shadows, each cast by its own fill of the same shape. Shadow
    // offsets use the unflipped base space: a negative height falls below.
    [NSGraphicsContext saveGraphicsState];
    NSShadow *wide = [NSShadow new];
    wide.shadowColor = srgb(30 / 255.0, 32 / 255.0, 31 / 255.0, 26 / 255.0);
    wide.shadowBlurRadius = 8;
    wide.shadowOffset = NSMakeSize(0, -4);
    [wide set];
    [background setFill];
    [outline fill];
    [NSGraphicsContext restoreGraphicsState];
    [NSGraphicsContext saveGraphicsState];
    NSShadow *tight = [NSShadow new];
    tight.shadowColor = srgb(24 / 255.0, 26 / 255.0, 24 / 255.0, 15 / 255.0);
    tight.shadowBlurRadius = 3;
    tight.shadowOffset = NSMakeSize(0, -2);
    [tight set];
    [background setFill];
    [outline fill];
    [NSGraphicsContext restoreGraphicsState];

    NSBezierPath *border = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(bar, kBorderWidth / 2, kBorderWidth / 2)
                                                           xRadius:kOuterRadius - kBorderWidth / 2
                                                           yRadius:kOuterRadius - kBorderWidth / 2];
    border.lineWidth = kBorderWidth;
    [(dark ? srgb(1, 1, 1, 0.2) : srgb(0, 0, 0, 0.2)) setStroke];
    [border stroke];

    NSColor *text = dark ? srgb(1, 1, 1, 0.8) : srgb(29 / 255.0, 29 / 255.0, 29 / 255.0, 1);
    NSColor *number = [text colorWithAlphaComponent:dark ? 0.45 : 0.5];
    NSColor *selection = dark ? srgb(0, 0.6, 0.949, 1) : srgb(0, 0.533, 1, 1);
    NSFont *bodyFont = self.bodyFont, *numberFont = self.numberFont;
    for (NSUInteger index = 0; index < self.itemRects.count; index++) {
        NSRect item = self.itemRects[index].rectValue;
        BOOL hot = index == self.highlighted;
        if (hot) {
            [selection setFill];
            [[NSBezierPath bezierPathWithRoundedRect:item xRadius:kSelectionRadius yRadius:kSelectionRadius] fill];
        }
        NSString *label = [NSString stringWithFormat:@"%lu", (unsigned long)(index + 1)];
        NSDictionary *numberStyle = @{NSFontAttributeName: numberFont,
                                      NSForegroundColorAttributeName: hot ? srgb(1, 1, 1, 0.8) : number};
        NSDictionary *bodyStyle = @{NSFontAttributeName: bodyFont,
                                    NSForegroundColorAttributeName: hot ? srgb(1, 1, 1, 1) : text};
        NSSize numberSize = [label sizeWithAttributes:numberStyle];
        NSSize bodySize = [self.candidates[index] sizeWithAttributes:bodyStyle];
        CGFloat x = NSMinX(item) + kSelectionPadding;
        [label drawAtPoint:NSMakePoint(x + (kNumberWidth - numberSize.width) / 2,
                                       NSMidY(item) - numberSize.height / 2) withAttributes:numberStyle];
        [self.candidates[index] drawAtPoint:NSMakePoint(x + kNumberWidth + kNumberTextSpacing,
                                                        NSMidY(item) - bodySize.height / 2) withAttributes:bodyStyle];
    }

    if (self.paged) {
        // Outlined chevron, as Chatterfly's pageButtonIconStyle.
        NSRect arrow = self.arrowRect;
        NSBezierPath *chevron = [NSBezierPath bezierPath];
        [chevron moveToPoint:NSMakePoint(NSMidX(arrow) - 4.5, NSMidY(arrow) - 2)];
        [chevron lineToPoint:NSMakePoint(NSMidX(arrow), NSMidY(arrow) + 2.5)];
        [chevron lineToPoint:NSMakePoint(NSMidX(arrow) + 4.5, NSMidY(arrow) - 2)];
        chevron.lineWidth = 1.5;
        chevron.lineCapStyle = NSLineCapStyleRound;
        chevron.lineJoinStyle = NSLineJoinStyleRound;
        [[text colorWithAlphaComponent:0.55] setStroke];
        [chevron stroke];
    }
}

- (void)mouseDown:(NSEvent *)event {
    NSPoint point = [self convertPoint:event.locationInWindow fromView:nil];
    if (self.paged && NSPointInRect(point, self.arrowRect)) {
        if (self.clickedArrow) self.clickedArrow();
        return;
    }
    for (NSUInteger index = 0; index < self.itemRects.count; index++) {
        if (NSPointInRect(point, self.itemRects[index].rectValue)) {
            if (self.clicked) self.clicked((NSInteger)index);
            return;
        }
    }
}
@end

@interface ArionCandidateWindow : NSPanel
@end

@implementation ArionCandidateWindow
- (BOOL)canBecomeKeyWindow { return NO; }
- (BOOL)canBecomeMainWindow { return NO; }
@end

@interface ArionCandidatePanel ()
@property(strong) ArionCandidateWindow *window;
@property(strong) ArionCandidateView *view;
@property NSUInteger revision;
@end

@implementation ArionCandidatePanel
- (instancetype)init {
    if (!(self = [super init])) return nil;
    self.window = [[ArionCandidateWindow alloc] initWithContentRect:NSZeroRect
        styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
          backing:NSBackingStoreBuffered defer:YES];
    self.window.opaque = NO;
    self.window.backgroundColor = NSColor.clearColor;
    self.window.hasShadow = NO;
    self.window.hidesOnDeactivate = NO;
    self.window.releasedWhenClosed = NO;
    self.window.level = CGWindowLevelForKey(kCGCursorWindowLevelKey) - 2;
    self.window.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces |
        NSWindowCollectionBehaviorFullScreenAuxiliary | NSWindowCollectionBehaviorStationary |
        NSWindowCollectionBehaviorIgnoresCycle;
    self.view = [ArionCandidateView new];
    self.window.contentView = self.view;
    __weak ArionCandidatePanel *weakSelf = self;
    self.view.clicked = ^(NSInteger index) {
        ArionCandidatePanel *panel = weakSelf;
        if (panel.selected) panel.selected((NSUInteger)index, panel.revision);
    };
    self.view.clickedArrow = ^{
        ArionCandidatePanel *panel = weakSelf;
        if (panel.nextPage) panel.nextPage(panel.revision);
    };
    return self;
}

- (void)showCandidates:(NSArray<NSString *> *)candidates
           highlighted:(NSUInteger)highlighted
              lastPage:(BOOL)lastPage
              revision:(NSUInteger)revision
             caretRect:(NSRect)caret {
    if (candidates.count == 0) { [self hide]; return; }
    self.revision = revision;
    self.view.candidates = candidates;
    self.view.highlighted = highlighted;
    // The chevron is the next-page affordance; earlier pages stay on keys.
    self.view.paged = !lastPage;
    NSSize bar = [self.view layoutBar];
    NSSize size = NSMakeSize(bar.width + 2 * kShadowMargin, bar.height + 2 * kShadowMargin);

    NSScreen *screen = NSScreen.mainScreen;
    for (NSScreen *candidate in NSScreen.screens) {
        if (NSPointInRect(NSMakePoint(NSMinX(caret), NSMidY(caret)), candidate.frame)) { screen = candidate; break; }
    }
    NSRect visible = screen.visibleFrame;
    // Align the first candidate's number with the caret, below its line;
    // above it when the bar would leave the screen.
    CGFloat x = NSMinX(caret) - kShadowMargin - kContentInset - kSelectionPadding;
    CGFloat y = NSMinY(caret) - kCaretGap - bar.height - kShadowMargin;
    if (y + kShadowMargin < NSMinY(visible)) y = NSMaxY(caret) + kCaretGap - kShadowMargin;
    x = MAX(NSMinX(visible) - kShadowMargin, MIN(x, NSMaxX(visible) + kShadowMargin - size.width));
    [self.window setFrame:NSMakeRect(x, y, size.width, size.height) display:NO];
    self.view.needsDisplay = YES;
    [self.window orderFront:nil];
}

- (void)hide {
    [self.window orderOut:nil];
}
@end
