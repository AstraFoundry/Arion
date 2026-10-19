#import "candidate_panel.h"
#import <QuartzCore/QuartzCore.h>

// Chatterfly (Yuanbao profile) size levels 1–3, read from
// +[YuanbaoProductProfile yuanbaoCandidateSizePresets].
typedef struct {
    CGFloat barHeight, selectionHeight, bodyFont, numberFont, numberWidth, selectionRadius, outerRadius;
} Preset;
static const Preset kPresets[] = {
    {30, 22, 14, 12, 8, 6, 10},
    {32, 24, 16, 13, 8, 6, 10},
    {35, 27, 18, 14, 10, 8, 12},
};
// Values shared by every level.
static const CGFloat kBorderWidth = 0.5;
static const CGFloat kSelectionPadding = 4;
static const CGFloat kContentInset = 4;
static const CGFloat kCandidateSpacing = 23;
static const CGFloat kNumberTextSpacing = 3;
static const CGFloat kArrowSize = 20;
static const CGFloat kArrowTrailingInset = 8;
// Room for the layer shadows (radius 8, offset 4) outside the bar.
static const CGFloat kShadowMargin = 14;
// Distance between the caret line and the bar.
static const CGFloat kCaretGap = 4;
// Resizing a window reallocates its backing store, which dominated per-key
// cost; the window only grows (with slack) while shown, and the transparent
// remainder right of the bar passes clicks through.
static const CGFloat kWidthSlack = 96;
// Ordering a window in costs far more than any per-key update, and a hidden
// bar returns with the next word. Hiding only makes it transparent and
// click-through; it leaves the window list after this idle time.
static const NSTimeInterval kOrderOutDelay = 2;

static NSColor *srgb(CGFloat r, CGFloat g, CGFloat b, CGFloat a) {
    return [NSColor colorWithSRGBRed:r green:g blue:b alpha:a];
}

// Draws the numbers, candidate text and chevron over the layer chrome.
@interface ArionCandidateText : NSView
@property(copy) NSArray<NSString *> *candidates;
@property(copy) NSArray<NSValue *> *itemRects;
@property(copy) NSArray<NSNumber *> *textWidths;
@property NSUInteger highlighted;
@property BOOL paged;
@property NSRect arrowRect;
@property Preset preset;
@property(strong) NSFont *bodyFont;
@property(strong) NSFont *numberFont;
@property BOOL dark;
@property(copy) void (^clicked)(NSInteger index);
@property(copy) void (^clickedArrow)(void);
@end

@implementation ArionCandidateText
- (BOOL)isFlipped { return YES; }
- (BOOL)isOpaque { return NO; }
- (BOOL)acceptsFirstMouse:(NSEvent *)event { return YES; }

- (void)drawRect:(NSRect)dirty {
    NSColor *text = self.dark ? srgb(1, 1, 1, 0.8) : srgb(29 / 255.0, 29 / 255.0, 29 / 255.0, 1);
    NSColor *number = [text colorWithAlphaComponent:self.dark ? 0.45 : 0.5];
    NSDictionary *numberStyle = @{NSFontAttributeName: self.numberFont, NSForegroundColorAttributeName: number};
    NSDictionary *hotNumberStyle = @{NSFontAttributeName: self.numberFont, NSForegroundColorAttributeName: srgb(1, 1, 1, 0.8)};
    NSDictionary *bodyStyle = @{NSFontAttributeName: self.bodyFont, NSForegroundColorAttributeName: text};
    NSDictionary *hotBodyStyle = @{NSFontAttributeName: self.bodyFont, NSForegroundColorAttributeName: srgb(1, 1, 1, 1)};
    CGFloat numberHeight = self.numberFont.ascender - self.numberFont.descender;
    CGFloat bodyHeight = self.bodyFont.ascender - self.bodyFont.descender;
    for (NSUInteger index = 0; index < self.itemRects.count; index++) {
        NSRect item = self.itemRects[index].rectValue;
        BOOL hot = index == self.highlighted;
        NSString *label = [NSString stringWithFormat:@"%lu", (unsigned long)(index + 1)];
        NSDictionary *style = hot ? hotNumberStyle : numberStyle;
        CGFloat x = NSMinX(item) + kSelectionPadding;
        CGFloat numberWidth = [label sizeWithAttributes:style].width;
        [label drawAtPoint:NSMakePoint(x + (self.preset.numberWidth - numberWidth) / 2, NSMidY(item) - numberHeight / 2)
            withAttributes:style];
        [self.candidates[index] drawAtPoint:NSMakePoint(x + self.preset.numberWidth + kNumberTextSpacing, NSMidY(item) - bodyHeight / 2)
                             withAttributes:hot ? hotBodyStyle : bodyStyle];
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

// Hosts the chrome layers; geometry is symmetric about the bar's center, so
// the unflipped layer space only matters for the downward shadow offsets.
@interface ArionCandidateView : NSView
@property(strong) CALayer *wideShadow;
@property(strong) CALayer *tightShadow;
@property(strong) CALayer *bar;
@property(strong) CALayer *selection;
@property(strong) ArionCandidateText *text;
@end

@implementation ArionCandidateView
- (instancetype)initWithFrame:(NSRect)frame {
    if (!(self = [super initWithFrame:frame])) return nil;
    self.wantsLayer = YES;
    self.layer = [CALayer layer];
    self.wideShadow = [CALayer layer];
    self.tightShadow = [CALayer layer];
    self.bar = [CALayer layer];
    self.selection = [CALayer layer];
    for (CALayer *shadow in @[self.wideShadow, self.tightShadow]) shadow.shadowOpacity = 1;
    self.wideShadow.shadowRadius = 8;
    self.wideShadow.shadowOffset = CGSizeMake(0, -4);
    self.wideShadow.shadowColor = srgb(30 / 255.0, 32 / 255.0, 31 / 255.0, 26 / 255.0).CGColor;
    self.tightShadow.shadowRadius = 3;
    self.tightShadow.shadowOffset = CGSizeMake(0, -2);
    self.tightShadow.shadowColor = srgb(24 / 255.0, 26 / 255.0, 24 / 255.0, 15 / 255.0).CGColor;
    self.bar.borderWidth = kBorderWidth;
    for (CALayer *layer in @[self.wideShadow, self.tightShadow, self.bar, self.selection]) [self.layer addSublayer:layer];
    self.text = [[ArionCandidateText alloc] initWithFrame:self.bounds];
    self.text.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
    [self addSubview:self.text];
    return self;
}

- (BOOL)dark {
    return [[self.effectiveAppearance bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]]
        isEqualToString:NSAppearanceNameDarkAqua];
}

// Lays out the page and returns the window size, shadow margin included.
- (NSSize)layoutCandidates:(NSArray<NSString *> *)candidates highlighted:(NSUInteger)highlighted
                     paged:(BOOL)paged preset:(Preset)preset {
    ArionCandidateText *text = self.text;
    if (text.preset.bodyFont != preset.bodyFont || !text.bodyFont) {
        text.bodyFont = [NSFont fontWithName:@"PingFangSC-Regular" size:preset.bodyFont] ?: [NSFont systemFontOfSize:preset.bodyFont];
        text.numberFont = [NSFont monospacedDigitSystemFontOfSize:preset.numberFont weight:NSFontWeightRegular];
    }
    text.preset = preset;
    NSDictionary *body = @{NSFontAttributeName: text.bodyFont};
    NSMutableArray<NSValue *> *rects = [NSMutableArray arrayWithCapacity:candidates.count];
    CGFloat x = kShadowMargin + kContentInset;
    CGFloat top = kShadowMargin + (preset.barHeight - preset.selectionHeight) / 2;
    CGFloat gap = kCandidateSpacing - 2 * kSelectionPadding;
    for (NSString *candidate in candidates) {
        CGFloat width = kSelectionPadding + preset.numberWidth + kNumberTextSpacing +
            ceil([candidate sizeWithAttributes:body].width) + kSelectionPadding;
        [rects addObject:[NSValue valueWithRect:NSMakeRect(x, top, width, preset.selectionHeight)]];
        x += width + gap;
    }
    x -= gap;
    if (paged) {
        x += kArrowTrailingInset;
        text.arrowRect = NSMakeRect(x, kShadowMargin + (preset.barHeight - kArrowSize) / 2, kArrowSize, kArrowSize);
        x += kArrowSize + kArrowTrailingInset;
    } else {
        x += kContentInset;
    }
    text.candidates = candidates;
    text.itemRects = rects;
    text.highlighted = highlighted;
    text.paged = paged;
    BOOL dark = self.dark;
    text.dark = dark;
    text.needsDisplay = YES;

    NSSize size = NSMakeSize(x + kShadowMargin, preset.barHeight + 2 * kShadowMargin);
    CGRect bar = CGRectMake(kShadowMargin, kShadowMargin, size.width - 2 * kShadowMargin, preset.barHeight);
    CGColorRef background = (dark ? srgb(23 / 255.0, 23 / 255.0, 23 / 255.0, 1) : srgb(1, 1, 1, 1)).CGColor;
    [CATransaction begin];
    [CATransaction setDisableActions:YES];
    CGPathRef outline = CGPathCreateWithRoundedRect(CGRectMake(0, 0, bar.size.width, bar.size.height),
                                                    preset.outerRadius, preset.outerRadius, NULL);
    for (CALayer *shadow in @[self.wideShadow, self.tightShadow]) {
        shadow.frame = bar;
        shadow.shadowPath = outline;
    }
    CGPathRelease(outline);
    self.bar.frame = bar;
    self.bar.cornerRadius = preset.outerRadius;
    self.bar.backgroundColor = background;
    self.bar.borderColor = (dark ? srgb(1, 1, 1, 0.2) : srgb(0, 0, 0, 0.2)).CGColor;
    // Layer space is unflipped; the selection is vertically centered.
    NSRect item = highlighted < rects.count ? rects[highlighted].rectValue : NSZeroRect;
    self.selection.hidden = highlighted >= rects.count;
    self.selection.frame = CGRectMake(NSMinX(item), size.height - NSMaxY(item), NSWidth(item), NSHeight(item));
    self.selection.cornerRadius = preset.selectionRadius;
    self.selection.backgroundColor = (dark ? srgb(0, 0.6, 0.949, 1) : srgb(0, 0.533, 1, 1)).CGColor;
    [CATransaction commit];
    return size;
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
@property BOOL shown;
@property NSUInteger hideGeneration;
@end

@implementation ArionCandidatePanel
- (instancetype)init {
    if (!(self = [super init])) return nil;
    _size = ArionCandidateSizeMedium;
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
    self.view = [[ArionCandidateView alloc] initWithFrame:NSZeroRect];
    self.window.contentView = self.view;
    __weak ArionCandidatePanel *weakSelf = self;
    self.view.text.clicked = ^(NSInteger index) {
        ArionCandidatePanel *panel = weakSelf;
        if (panel.selected) panel.selected((NSUInteger)index, panel.revision);
    };
    self.view.text.clickedArrow = ^{
        ArionCandidatePanel *panel = weakSelf;
        if (panel.nextPage) panel.nextPage(panel.revision);
    };
    return self;
}

- (void)setAppearance:(ArionCandidateAppearance)appearance {
    _appearance = appearance;
    self.window.appearance = appearance == ArionCandidateAppearanceLight ? [NSAppearance appearanceNamed:NSAppearanceNameAqua] :
        appearance == ArionCandidateAppearanceDark ? [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua] : nil;
}

- (void)setSize:(ArionCandidateSize)size {
    _size = MAX(ArionCandidateSizeSmall, MIN(ArionCandidateSizeLarge, size));
}

- (void)showCandidates:(NSArray<NSString *> *)candidates
           highlighted:(NSUInteger)highlighted
              lastPage:(BOOL)lastPage
              revision:(NSUInteger)revision
             caretRect:(NSRect)caret {
    if (candidates.count == 0) { [self hide]; return; }
    self.revision = revision;
    // The chevron is the next-page affordance; earlier pages stay on keys.
    NSSize size = [self.view layoutCandidates:candidates highlighted:highlighted paged:!lastPage preset:kPresets[self.size]];
    CGFloat barHeight = size.height - 2 * kShadowMargin;

    NSScreen *screen = NSScreen.mainScreen;
    for (NSScreen *candidate in NSScreen.screens) {
        if (NSPointInRect(NSMakePoint(NSMinX(caret), NSMidY(caret)), candidate.frame)) { screen = candidate; break; }
    }
    NSRect visible = screen.visibleFrame;
    // Align the first candidate's number with the caret, below its line;
    // above it when the bar would leave the screen.
    CGFloat x = NSMinX(caret) - kShadowMargin - kContentInset - kSelectionPadding;
    CGFloat y = NSMinY(caret) - kCaretGap - barHeight - kShadowMargin;
    if (y + kShadowMargin < NSMinY(visible)) y = NSMaxY(caret) + kCaretGap - kShadowMargin;
    x = MAX(NSMinX(visible) - kShadowMargin, MIN(x, NSMaxX(visible) + kShadowMargin - size.width));
    NSRect current = self.window.frame;
    BOOL fits = self.shown && NSHeight(current) == size.height && NSWidth(current) >= size.width;
    NSRect frame = NSMakeRect(x, y, fits ? NSWidth(current) : size.width + kWidthSlack, size.height);
    if (!NSEqualSizes(frame.size, current.size)) [self.window setFrame:frame display:NO];
    else if (!NSEqualPoints(frame.origin, current.origin)) [self.window setFrameOrigin:frame.origin];
    if (!self.window.visible) [self.window orderFront:nil];
    if (!self.shown) {
        self.shown = YES;
        self.hideGeneration++;
        self.window.alphaValue = 1;
        self.window.ignoresMouseEvents = NO;
    }
}

- (void)hide {
    if (!self.shown) return;
    self.shown = NO;
    self.window.alphaValue = 0;
    self.window.ignoresMouseEvents = YES;
    NSUInteger generation = ++self.hideGeneration;
    __weak ArionCandidatePanel *weakSelf = self;
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(kOrderOutDelay * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
        ArionCandidatePanel *panel = weakSelf;
        if (panel && !panel.shown && panel.hideGeneration == generation) [panel.window orderOut:nil];
    });
}
@end
