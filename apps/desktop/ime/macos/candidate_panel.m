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
// -[SGCandidatesController updateWindow:] animates spreading over 0.3 s and
// folding over 0.2 s with the default ease-in-out curve.
static const NSTimeInterval kExpandDuration = 0.3;
static const NSTimeInterval kCollapseDuration = 0.2;
// The grid's rows fade in once the bar has started to grow.
static const NSTimeInterval kRowsFadeDelay = 0.08;
static const NSTimeInterval kRowsFadeDuration = 0.2;

static NSColor *srgb(CGFloat r, CGFloat g, CGFloat b, CGFloat a) {
    return [NSColor colorWithSRGBRed:r green:g blue:b alpha:a];
}

// Draws the numbers, candidate text and chevron over the layer chrome, in
// flipped window coordinates.
@interface ArionCandidateText : NSView
@property(copy) NSArray<NSArray<NSString *> *> *rows;
@property(copy) NSArray<NSArray<NSValue *> *> *itemRects;
@property NSUInteger currentRow;
@property NSUInteger highlighted;
@property BOOL expanded;
@property BOOL chevron;
@property NSRect arrowRect;
@property Preset preset;
@property(strong) NSFont *bodyFont;
@property(strong) NSFont *numberFont;
@property BOOL dark;
// The highlighted candidate's text, on the accent fill.
@property(strong) NSColor *selectedText;
@property(copy) void (^clicked)(NSUInteger index, NSUInteger row);
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
    NSDictionary *hotNumberStyle = @{NSFontAttributeName: self.numberFont,
                                     NSForegroundColorAttributeName: [self.selectedText colorWithAlphaComponent:0.8]};
    NSDictionary *bodyStyle = @{NSFontAttributeName: self.bodyFont, NSForegroundColorAttributeName: text};
    NSDictionary *hotBodyStyle = @{NSFontAttributeName: self.bodyFont, NSForegroundColorAttributeName: self.selectedText};
    CGFloat numberHeight = self.numberFont.ascender - self.numberFont.descender;
    CGFloat bodyHeight = self.bodyFont.ascender - self.bodyFont.descender;
    for (NSUInteger row = 0; row < self.rows.count; row++) {
        BOOL current = row == self.currentRow;
        for (NSUInteger index = 0; index < self.rows[row].count; index++) {
            NSRect item = self.itemRects[row][index].rectValue;
            BOOL hot = current && index == self.highlighted;
            CGFloat x = NSMinX(item) + kSelectionPadding;
            // Number keys act on the current row only, so only it is numbered.
            if (current) {
                NSString *label = [NSString stringWithFormat:@"%lu", (unsigned long)(index + 1)];
                NSDictionary *style = hot ? hotNumberStyle : numberStyle;
                CGFloat width = [label sizeWithAttributes:style].width;
                [label drawAtPoint:NSMakePoint(x + (self.preset.numberWidth - width) / 2, NSMidY(item) - numberHeight / 2)
                    withAttributes:style];
            }
            [self.rows[row][index] drawAtPoint:NSMakePoint(x + self.preset.numberWidth + kNumberTextSpacing, NSMidY(item) - bodyHeight / 2)
                                withAttributes:hot ? hotBodyStyle : bodyStyle];
        }
    }
    if (self.chevron) {
        // Outlined chevron, as Chatterfly's pageButtonIconStyle: down to
        // expand, up to collapse.
        NSRect arrow = self.arrowRect;
        CGFloat tip = self.expanded ? -2.5 : 2.5;
        NSBezierPath *chevron = [NSBezierPath bezierPath];
        [chevron moveToPoint:NSMakePoint(NSMidX(arrow) - 4.5, NSMidY(arrow) - tip * 0.8)];
        [chevron lineToPoint:NSMakePoint(NSMidX(arrow), NSMidY(arrow) + tip)];
        [chevron lineToPoint:NSMakePoint(NSMidX(arrow) + 4.5, NSMidY(arrow) - tip * 0.8)];
        chevron.lineWidth = 1.5;
        chevron.lineCapStyle = NSLineCapStyleRound;
        chevron.lineJoinStyle = NSLineJoinStyleRound;
        [[text colorWithAlphaComponent:0.55] setStroke];
        [chevron stroke];
    }
}

- (void)mouseDown:(NSEvent *)event {
    NSPoint point = [self convertPoint:event.locationInWindow fromView:nil];
    // The window is already the final size while the bar is still opening or
    // closing. Hits outside the presented bar would select a row that is not
    // drawn yet.
    CALayer *mask = self.layer.mask.presentationLayer ?: self.layer.mask;
    if (mask && !CGRectContainsPoint(mask.frame, point)) return;
    if (self.chevron && NSPointInRect(point, NSInsetRect(self.arrowRect, -4, -4))) {
        if (self.clickedArrow) self.clickedArrow();
        return;
    }
    for (NSUInteger row = 0; row < self.itemRects.count; row++) {
        for (NSUInteger index = 0; index < self.itemRects[row].count; index++) {
            if (NSPointInRect(point, self.itemRects[row][index].rectValue)) {
                if (self.clicked) self.clicked(index, row);
                return;
            }
        }
    }
}
@end

// Hosts the chrome layers in unflipped layer space; the text view above
// them uses flipped coordinates of the same window.
@interface ArionCandidateView : NSView
@property(strong) CALayer *wideShadow;
@property(strong) CALayer *tightShadow;
@property(strong) CALayer *bar;
@property(strong) CALayer *rowTint;
@property(strong) CAShapeLayer *separators;
@property(strong) CALayer *selection;
// Clips the text to the bar, so rows appear inside it as it grows.
@property(strong) CALayer *textMask;
@property(strong) ArionCandidateText *text;
@property(strong) NSColor *lightFill;
@property(strong) NSColor *lightText;
@property(strong) NSColor *darkFill;
@property(strong) NSColor *darkText;
// The bar's last frame, where an expand or collapse animation starts.
@property CGRect barFrame;
@end

@implementation ArionCandidateView
- (instancetype)initWithFrame:(NSRect)frame {
    if (!(self = [super initWithFrame:frame])) return nil;
    self.wantsLayer = YES;
    self.layer = [CALayer layer];
    self.wideShadow = [CALayer layer];
    self.tightShadow = [CALayer layer];
    self.bar = [CALayer layer];
    self.rowTint = [CALayer layer];
    self.separators = [CAShapeLayer layer];
    self.selection = [CALayer layer];
    for (CALayer *shadow in @[self.wideShadow, self.tightShadow]) shadow.shadowOpacity = 1;
    self.wideShadow.shadowRadius = 8;
    self.wideShadow.shadowOffset = CGSizeMake(0, -4);
    self.wideShadow.shadowColor = srgb(30 / 255.0, 32 / 255.0, 31 / 255.0, 26 / 255.0).CGColor;
    self.tightShadow.shadowRadius = 3;
    self.tightShadow.shadowOffset = CGSizeMake(0, -2);
    self.tightShadow.shadowColor = srgb(24 / 255.0, 26 / 255.0, 24 / 255.0, 15 / 255.0).CGColor;
    self.bar.borderWidth = kBorderWidth;
    self.separators.lineWidth = kBorderWidth;
    self.separators.fillColor = NULL;
    for (CALayer *layer in @[self.wideShadow, self.tightShadow, self.bar, self.rowTint, self.separators, self.selection]) {
        [self.layer addSublayer:layer];
    }
    self.text = [[ArionCandidateText alloc] initWithFrame:self.bounds];
    self.text.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
    self.text.wantsLayer = YES;
    [self addSubview:self.text];
    self.textMask = [CALayer layer];
    self.textMask.backgroundColor = NSColor.blackColor.CGColor;
    self.text.layer.mask = self.textMask;
    return self;
}

- (BOOL)dark {
    return [[self.effectiveAppearance bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]]
        isEqualToString:NSAppearanceNameDarkAqua];
}

// Measures the content (bar) size for rows without changing anything.
- (NSSize)contentSizeForRows:(NSArray<NSArray<NSString *> *> *)rows expanded:(BOOL)expanded
                     chevron:(BOOL)chevron preset:(Preset)preset widths:(NSArray<NSNumber *> **)columns {
    [self prepareFonts:preset];
    NSDictionary *body = @{NSFontAttributeName: self.text.bodyFont};
    CGFloat cell = kSelectionPadding + preset.numberWidth + kNumberTextSpacing + kSelectionPadding;
    NSMutableArray<NSNumber *> *widths = [NSMutableArray array];
    // Collapsed, each candidate is as wide as its text; expanded, a column is
    // as wide as its widest candidate, so the rows line up.
    NSArray<NSArray<NSString *> *> *measured = expanded ? rows : @[rows.firstObject ?: @[]];
    for (NSArray<NSString *> *row in measured) {
        for (NSUInteger i = 0; i < row.count; i++) {
            CGFloat width = cell + ceil([row[i] sizeWithAttributes:body].width);
            if (!expanded) { [widths addObject:@(width)]; continue; }
            if (i < widths.count) widths[i] = @(MAX(widths[i].doubleValue, width));
            else [widths addObject:@(width)];
        }
    }
    CGFloat gap = kCandidateSpacing - 2 * kSelectionPadding;
    CGFloat width = kContentInset;
    for (NSNumber *column in widths) width += column.doubleValue + gap;
    width -= widths.count ? gap : 0;
    width += chevron ? kArrowTrailingInset + kArrowSize + kArrowTrailingInset : kContentInset;
    if (columns) *columns = widths;
    CGFloat height = preset.barHeight * (expanded ? rows.count : 1);
    // Partial choices can leave fewer rows. Keep the expanded viewport until
    // the user folds it or the composition hides the panel.
    if (expanded) height = MAX(height, CGRectGetHeight(self.barFrame));
    return NSMakeSize(width, height);
}

- (void)prepareFonts:(Preset)preset {
    ArionCandidateText *text = self.text;
    if (text.preset.bodyFont != preset.bodyFont || !text.bodyFont) {
        text.bodyFont = [NSFont fontWithName:@"PingFangSC-Regular" size:preset.bodyFont] ?: [NSFont systemFontOfSize:preset.bodyFont];
        text.numberFont = [NSFont monospacedDigitSystemFontOfSize:preset.numberFont weight:NSFontWeightRegular];
    }
    text.preset = preset;
}

// Lays out rows in a canvas of the window's size. The bar hangs from the
// canvas top (anchorTop) or stands on its bottom, inside the shadow margin.
// animation is 0 for none, else the expand or collapse duration.
- (void)layoutRows:(NSArray<NSArray<NSString *> *> *)rows currentRow:(NSUInteger)currentRow highlighted:(NSUInteger)highlighted
          expanded:(BOOL)expanded chevron:(BOOL)chevron preset:(Preset)preset canvas:(NSSize)canvas
         anchorTop:(BOOL)anchorTop animation:(NSTimeInterval)animation {
    NSArray<NSNumber *> *columns = nil;
    NSSize content = [self contentSizeForRows:rows expanded:expanded chevron:chevron preset:preset widths:&columns];
    ArionCandidateText *text = self.text;
    // Flipped (text) space: top of the bar.
    CGFloat top = anchorTop ? kShadowMargin : canvas.height - kShadowMargin - content.height;
    NSArray<NSArray<NSString *> *> *shown = expanded ? rows : @[rows.firstObject ?: @[]];
    NSUInteger current = expanded ? currentRow : 0;
    CGFloat gap = kCandidateSpacing - 2 * kSelectionPadding;
    NSMutableArray<NSArray<NSValue *> *> *rects = [NSMutableArray array];
    for (NSUInteger row = 0; row < shown.count; row++) {
        NSMutableArray<NSValue *> *items = [NSMutableArray array];
        CGFloat x = kShadowMargin + kContentInset;
        CGFloat y = top + row * preset.barHeight + (preset.barHeight - preset.selectionHeight) / 2;
        for (NSUInteger i = 0; i < shown[row].count; i++) {
            CGFloat width = columns[i].doubleValue;
            [items addObject:[NSValue valueWithRect:NSMakeRect(x, y, width, preset.selectionHeight)]];
            x += width + gap;
        }
        [rects addObject:items];
    }
    CGFloat barRight = kShadowMargin + content.width;
    text.arrowRect = NSMakeRect(barRight - kArrowTrailingInset - kArrowSize, top + (preset.barHeight - kArrowSize) / 2, kArrowSize, kArrowSize);
    text.rows = shown;
    text.itemRects = rects;
    text.currentRow = current;
    text.highlighted = highlighted;
    text.expanded = expanded;
    text.chevron = chevron;
    BOOL dark = self.dark;
    text.dark = dark;
    // Without settings, Sunbeam's yellow (#FFCC00) with ink (#14110F).
    NSColor *fill = (dark ? self.darkFill : self.lightFill) ?: srgb(1, 0.8, 0, 1);
    text.selectedText = (dark ? self.darkText : self.lightText) ?: srgb(20 / 255.0, 17 / 255.0, 15 / 255.0, 1);
    text.needsDisplay = YES;

    // Unflipped (layer) space.
    CGFloat flip = canvas.height;
    CGRect bar = CGRectMake(kShadowMargin, flip - top - content.height, content.width, content.height);
    CGRect from = CGRectIsEmpty(self.barFrame) ? bar : self.barFrame;
    // A collapse or expand keeps the anchored edge where it was.
    if (animation > 0) from.origin.y = anchorTop ? NSMaxY(bar) - from.size.height : bar.origin.y;
    CGColorRef background = (dark ? srgb(23 / 255.0, 23 / 255.0, 23 / 255.0, 1) : srgb(1, 1, 1, 1)).CGColor;
    [CATransaction begin];
    [CATransaction setDisableActions:YES];
    CGPathRef outline = CGPathCreateWithRoundedRect(CGRectMake(0, 0, bar.size.width, bar.size.height),
                                                    preset.outerRadius, preset.outerRadius, NULL);
    CGPathRef fromOutline = CGPathCreateWithRoundedRect(CGRectMake(0, 0, from.size.width, from.size.height),
                                                        preset.outerRadius, preset.outerRadius, NULL);
    self.textMask.cornerRadius = preset.outerRadius;
    for (CALayer *layer in @[self.wideShadow, self.tightShadow, self.bar, self.textMask]) {
        // The text view's layer is flipped, so its mask uses flipped frames.
        BOOL flipped = layer == self.textMask;
        CGRect to = flipped ? CGRectMake(bar.origin.x, flip - CGRectGetMaxY(bar), bar.size.width, bar.size.height) : bar;
        CGRect start = flipped ? CGRectMake(from.origin.x, flip - CGRectGetMaxY(from), from.size.width, from.size.height) : from;
        layer.frame = to;
        if (layer != self.bar && layer != self.textMask) layer.shadowPath = outline;
        if (animation <= 0) continue;
        CAMediaTimingFunction *curve = [CAMediaTimingFunction functionWithName:kCAMediaTimingFunctionEaseInEaseOut];
        CABasicAnimation *bounds = [CABasicAnimation animationWithKeyPath:@"bounds"];
        bounds.fromValue = [NSValue valueWithRect:NSMakeRect(0, 0, from.size.width, from.size.height)];
        CABasicAnimation *position = [CABasicAnimation animationWithKeyPath:@"position"];
        position.fromValue = [NSValue valueWithPoint:NSMakePoint(CGRectGetMidX(start), CGRectGetMidY(start))];
        NSMutableArray *animations = [NSMutableArray arrayWithObjects:bounds, position, nil];
        if (layer != self.bar && layer != self.textMask) {
            CABasicAnimation *path = [CABasicAnimation animationWithKeyPath:@"shadowPath"];
            path.fromValue = (__bridge id)fromOutline;
            [animations addObject:path];
        }
        CAAnimationGroup *group = [CAAnimationGroup animation];
        group.animations = animations;
        group.duration = animation;
        group.timingFunction = curve;
        [layer addAnimation:group forKey:@"resize"];
    }
    CGPathRelease(outline);
    CGPathRelease(fromOutline);
    self.barFrame = bar;
    self.bar.cornerRadius = preset.outerRadius;
    self.bar.backgroundColor = background;
    self.bar.borderColor = (dark ? srgb(1, 1, 1, 0.2) : srgb(0, 0, 0, 0.2)).CGColor;

    // The current row of the grid is tinted; rows are separated by hairlines.
    CGFloat rowTop = flip - top - (current + 1) * preset.barHeight;
    self.rowTint.hidden = !expanded;
    self.rowTint.frame = CGRectMake(kShadowMargin + 2, rowTop + 2, content.width - 4, preset.barHeight - 4);
    self.rowTint.cornerRadius = preset.selectionRadius + 2;
    self.rowTint.backgroundColor = (dark ? srgb(1, 1, 1, 0.07) : srgb(0, 0, 0, 0.045)).CGColor;
    CGMutablePathRef lines = CGPathCreateMutable();
    for (NSUInteger row = 1; expanded && row < shown.count; row++) {
        CGFloat y = flip - top - row * preset.barHeight;
        CGPathMoveToPoint(lines, NULL, kShadowMargin + kContentInset + kSelectionPadding, y);
        CGPathAddLineToPoint(lines, NULL, kShadowMargin + content.width - kContentInset - kSelectionPadding, y);
    }
    self.separators.path = lines;
    CGPathRelease(lines);
    self.separators.strokeColor = (dark ? srgb(1, 1, 1, 0.08) : srgb(0, 0, 0, 0.06)).CGColor;

    NSArray<NSValue *> *currentItems = rects.count > current ? rects[current] : @[];
    NSRect item = highlighted < currentItems.count ? currentItems[highlighted].rectValue : NSZeroRect;
    self.selection.hidden = highlighted >= currentItems.count;
    self.selection.frame = CGRectMake(NSMinX(item), flip - NSMaxY(item), NSWidth(item), NSHeight(item));
    self.selection.cornerRadius = preset.selectionRadius;
    self.selection.backgroundColor = fill.CGColor;
    if (animation >= kExpandDuration) {
        // The growing mask reveals new rows. The current row's text and
        // highlight stay visible; only the new row chrome fades in.
        for (CALayer *layer in @[self.rowTint, self.separators]) {
            CABasicAnimation *fade = [CABasicAnimation animationWithKeyPath:@"opacity"];
            fade.fromValue = @0;
            fade.toValue = @1;
            fade.beginTime = CACurrentMediaTime() + kRowsFadeDelay;
            fade.duration = kRowsFadeDuration;
            fade.fillMode = kCAFillModeBackwards;
            [layer addAnimation:fade forKey:@"appear"];
        }
    }
    [CATransaction commit];
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
@property BOOL expanded;
@property NSUInteger hideGeneration;
// Bumped by every show; a finished collapse only shrinks the window if no
// later show replaced its state.
@property NSUInteger showGeneration;
@property(copy) void (^reshow)(void);
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
    self.view.text.clicked = ^(NSUInteger index, NSUInteger row) {
        ArionCandidatePanel *panel = weakSelf;
        if (panel.selected) panel.selected(index, row, panel.revision);
    };
    self.view.text.clickedArrow = ^{
        ArionCandidatePanel *panel = weakSelf;
        if (panel.toggled) panel.toggled(panel.revision);
    };
    return self;
}

- (void)setAppearance:(ArionCandidateAppearance)appearance {
    _appearance = appearance;
    self.window.appearance = appearance == ArionCandidateAppearanceLight ? [NSAppearance appearanceNamed:NSAppearanceNameAqua] :
        appearance == ArionCandidateAppearanceDark ? [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua] : nil;
}

- (void)setLightFill:(NSColor *)color { _lightFill = color; self.view.lightFill = color; }
- (void)setLightText:(NSColor *)color { _lightText = color; self.view.lightText = color; }
- (void)setDarkFill:(NSColor *)color { _darkFill = color; self.view.darkFill = color; }
- (void)setDarkText:(NSColor *)color { _darkText = color; self.view.darkText = color; }

- (void)setSize:(ArionCandidateSize)size {
    _size = MAX(ArionCandidateSizeSmall, MIN(ArionCandidateSizeLarge, size));
}

- (void)showCandidates:(NSArray<NSString *> *)candidates
           highlighted:(NSUInteger)highlighted
              lastPage:(BOOL)lastPage
              revision:(NSUInteger)revision
             caretRect:(NSRect)caret {
    [self showRows:@[candidates] currentRow:0 highlighted:highlighted expanded:NO more:!lastPage revision:revision caretRect:caret];
}

- (void)showRows:(NSArray<NSArray<NSString *> *> *)rows
      currentRow:(NSUInteger)currentRow
     highlighted:(NSUInteger)highlighted
        expanded:(BOOL)expanded
            more:(BOOL)more
        revision:(NSUInteger)revision
       caretRect:(NSRect)caret {
    if (rows.count == 0 || rows[MIN(currentRow, rows.count - 1)].count == 0) { [self hide]; return; }
    self.revision = revision;
    self.reshow = nil;
    NSUInteger generation = ++self.showGeneration;
    Preset preset = kPresets[self.size];
    BOOL chevron = expanded || more;
    NSSize content = [self.view contentSizeForRows:rows expanded:expanded chevron:chevron preset:preset widths:NULL];
    NSSize size = NSMakeSize(content.width + 2 * kShadowMargin, content.height + 2 * kShadowMargin);

    NSScreen *screen = NSScreen.mainScreen;
    for (NSScreen *candidate in NSScreen.screens) {
        if (NSPointInRect(NSMakePoint(NSMinX(caret), NSMidY(caret)), candidate.frame)) { screen = candidate; break; }
    }
    NSRect visible = screen.visibleFrame;
    // Align the first candidate's number with the caret. The bar hangs below
    // the caret line and grows downward; without room it stands above the
    // line and grows upward.
    CGFloat x = NSMinX(caret) - kShadowMargin - kContentInset - kSelectionPadding;
    CGFloat below = NSMinY(caret) - kCaretGap + kShadowMargin;
    BOOL anchorTop = below - size.height >= NSMinY(visible);
    CGFloat y = anchorTop ? below - size.height : NSMaxY(caret) + kCaretGap - kShadowMargin;
    x = MAX(NSMinX(visible) - kShadowMargin, MIN(x, NSMaxX(visible) + kShadowMargin - size.width));

    BOOL changing = self.shown && expanded != self.expanded;
    NSTimeInterval animation = changing ? (expanded ? kExpandDuration : kCollapseDuration) : 0;
    self.expanded = expanded;
    NSRect current = self.window.frame;
    BOOL fits = self.shown && NSHeight(current) == size.height && NSWidth(current) >= size.width;
    NSRect frame = NSMakeRect(x, y, fits ? NSWidth(current) : size.width + kWidthSlack, size.height);
    if (changing && !expanded) {
        // Collapsing: keep the tall window while the bar shrinks inside it,
        // then fit the window to the bar.
        frame = NSMakeRect(x, anchorTop ? NSMaxY(current) - NSHeight(current) : y, MAX(NSWidth(current), NSWidth(frame)), NSHeight(current));
        __weak ArionCandidatePanel *weakSelf = self;
        NSArray *saved = [rows copy];
        self.reshow = ^{
            ArionCandidatePanel *panel = weakSelf;
            if (!panel || panel.showGeneration != generation || !panel.shown) return;
            [panel showRows:saved currentRow:currentRow highlighted:highlighted expanded:NO more:more revision:revision caretRect:caret];
        };
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(kCollapseDuration * NSEC_PER_SEC)), dispatch_get_main_queue(), ^{
            ArionCandidatePanel *panel = weakSelf;
            if (!panel || panel.showGeneration != generation || !panel.reshow) return;
            void (^reshow)(void) = panel.reshow;
            panel.reshow = nil;
            reshow();
        });
    }
    if (!NSEqualSizes(frame.size, current.size)) [self.window setFrame:frame display:NO];
    else if (!NSEqualPoints(frame.origin, current.origin)) [self.window setFrameOrigin:frame.origin];
    [self.view layoutRows:rows currentRow:currentRow highlighted:highlighted expanded:expanded chevron:chevron preset:preset
                   canvas:frame.size anchorTop:anchorTop animation:animation];
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
    self.expanded = NO;
    self.reshow = nil;
    self.view.barFrame = CGRectZero;
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

// The badge turns over like a flip card when the mode changes: the previous
// label folds away around the horizontal axis, then the new one unfolds. The
// two halves fit well inside the badge's one-second lifetime.
static const CGFloat kBadgeSize = 23;
static const NSTimeInterval kFlipHalfDuration = 0.12;
// Eye distance for the flip's perspective, in points.
static const CGFloat kFlipPerspective = 60;

static CATextLayer *modeLabelLayer(NSFont *font) {
    CATextLayer *layer = [CATextLayer layer];
    CGFloat height = ceil(font.ascender - font.descender);
    layer.frame = CGRectMake(0, floor((kBadgeSize - height) / 2), kBadgeSize, height);
    layer.font = (__bridge CFTypeRef)font;
    layer.fontSize = font.pointSize;
    layer.foregroundColor = NSColor.whiteColor.CGColor;
    layer.alignmentMode = kCAAlignmentCenter;
    return layer;
}

@interface ArionLanguagePanel ()
@property(strong) ArionCandidateWindow *window;
// The selected mode, and the previous one while it folds away.
@property(strong) CATextLayer *label;
@property(strong) CATextLayer *outgoing;
@property(strong) NSTimer *dismissal;
@end

@implementation ArionLanguagePanel
- (instancetype)init {
    if (!(self = [super init])) return nil;
    self.window = [[ArionCandidateWindow alloc] initWithContentRect:NSMakeRect(0, 0, kBadgeSize, kBadgeSize)
        styleMask:NSWindowStyleMaskBorderless | NSWindowStyleMaskNonactivatingPanel
        backing:NSBackingStoreBuffered defer:YES];
    self.window.opaque = NO;
    self.window.backgroundColor = NSColor.clearColor;
    self.window.hasShadow = NO;
    self.window.animationBehavior = NSWindowAnimationBehaviorNone;
    self.window.ignoresMouseEvents = YES;
    self.window.hidesOnDeactivate = NO;
    self.window.releasedWhenClosed = NO;
    self.window.level = CGWindowLevelForKey(kCGCursorWindowLevelKey) - 2;
    self.window.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces |
        NSWindowCollectionBehaviorFullScreenAuxiliary | NSWindowCollectionBehaviorStationary |
        NSWindowCollectionBehaviorIgnoresCycle;
    NSView *surface = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, kBadgeSize, kBadgeSize)];
    surface.wantsLayer = YES;
    // Original vector badge; no packaged competitor artwork or material blur.
    surface.layer.backgroundColor = NSColor.blackColor.CGColor;
    surface.layer.borderColor = NSColor.darkGrayColor.CGColor;
    surface.layer.borderWidth = kBorderWidth;
    surface.layer.cornerRadius = 5;
    surface.layer.masksToBounds = YES;
    // The view owns its backing layer's geometry, so the perspective lives
    // on a plain sublayer.
    CALayer *card = [CALayer layer];
    card.frame = surface.bounds;
    CATransform3D perspective = CATransform3DIdentity;
    perspective.m34 = -1.0 / kFlipPerspective;
    card.sublayerTransform = perspective;
    NSFont *font = [NSFont systemFontOfSize:15 weight:NSFontWeightMedium];
    self.label = modeLabelLayer(font);
    self.outgoing = modeLabelLayer(font);
    self.outgoing.opacity = 0;
    [card addSublayer:self.outgoing];
    [card addSublayer:self.label];
    [surface.layer addSublayer:card];
    self.window.contentView = surface;
    return self;
}
- (void)setAppearance:(ArionCandidateAppearance)appearance {
    _appearance = appearance;
    self.window.appearance = appearance == ArionCandidateAppearanceLight ? [NSAppearance appearanceNamed:NSAppearanceNameAqua] :
        appearance == ArionCandidateAppearanceDark ? [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua] : nil;
}
- (void)show:(NSString *)label previous:(NSString *)previous caretRect:(NSRect)caret {
    [self hide];
    // A host that cannot supply a caret must not produce a stray screen-corner hint.
    if (NSHeight(caret) <= 0) return;
    NSScreen *screen = nil;
    for (NSScreen *candidate in NSScreen.screens) {
        if (NSPointInRect(NSMakePoint(NSMinX(caret), NSMidY(caret)), candidate.frame)) { screen = candidate; break; }
    }
    if (!screen) return;
    NSRect visible = screen.visibleFrame;
    NSSize size = self.window.frame.size;
    CGFloat x = MAX(NSMinX(visible), MIN(NSMinX(caret), NSMaxX(visible) - size.width));
    CGFloat y = NSMinY(caret) - kCaretGap - size.height;
    if (y < NSMinY(visible)) y = NSMaxY(caret) + kCaretGap;
    y = MAX(NSMinY(visible), MIN(y, NSMaxY(visible) - size.height));
    [self.window setFrameOrigin:NSMakePoint(x, y)];
    [self showLabel:label previous:previous scale:screen.backingScaleFactor];
    [self.window orderFrontRegardless];
    // The badge only appears for a switch, so it always turns over from the
    // state that was just left.
    if (!NSWorkspace.sharedWorkspace.accessibilityDisplayShouldReduceMotion) [self flip];
    __weak ArionLanguagePanel *weakSelf = self;
    self.dismissal = [NSTimer scheduledTimerWithTimeInterval:1.0 repeats:NO block:^(NSTimer *timer) {
        (void)timer;
        [weakSelf hide];
    }];
}
- (void)showLabel:(NSString *)label previous:(NSString *)previous scale:(CGFloat)scale {
    [CATransaction begin];
    [CATransaction setDisableActions:YES];
    for (CATextLayer *layer in @[self.label, self.outgoing]) {
        [layer removeAllAnimations];
        layer.contentsScale = scale;
    }
    self.label.string = label;
    self.outgoing.string = previous;
    [CATransaction commit];
}
- (void)flip {
    // The outgoing label stays transparent in the model; the animation shows
    // it only while it folds to edge-on.
    CABasicAnimation *fold = [CABasicAnimation animationWithKeyPath:@"transform.rotation.x"];
    fold.fromValue = @0;
    fold.toValue = @(M_PI_2);
    CABasicAnimation *shown = [CABasicAnimation animationWithKeyPath:@"opacity"];
    shown.fromValue = @1;
    shown.toValue = @1;
    CAAnimationGroup *away = [CAAnimationGroup animation];
    away.animations = @[fold, shown];
    away.duration = kFlipHalfDuration;
    away.timingFunction = [CAMediaTimingFunction functionWithName:kCAMediaTimingFunctionEaseIn];
    [self.outgoing addAnimation:away forKey:@"flip"];
    // The new label waits edge-on, then unfolds into place.
    CABasicAnimation *unfold = [CABasicAnimation animationWithKeyPath:@"transform.rotation.x"];
    unfold.fromValue = @(-M_PI_2);
    unfold.toValue = @0;
    unfold.beginTime = [self.label convertTime:CACurrentMediaTime() fromLayer:nil] + kFlipHalfDuration;
    unfold.duration = kFlipHalfDuration;
    unfold.fillMode = kCAFillModeBackwards;
    unfold.timingFunction = [CAMediaTimingFunction functionWithName:kCAMediaTimingFunctionEaseOut];
    [self.label addAnimation:unfold forKey:@"flip"];
}
- (void)hide {
    [self.dismissal invalidate];
    self.dismissal = nil;
    [self.window orderOut:nil];
}
- (void)dealloc { [_dismissal invalidate]; }
@end
