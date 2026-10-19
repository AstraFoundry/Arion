#import <Cocoa/Cocoa.h>
#import "keyboard.h"

// Owns keyboard composition independently of the asynchronous voice draft.
@interface ArionTyping : NSObject
@property(readonly) BOOL composing;
@property(readonly) BOOL ascii;
@property(readonly) BOOL available;
@property(readonly, copy) NSString *recoveryText;
@property(readonly) NSUInteger revision;
@property(readonly, strong) ArionKeyboardSnapshot *snapshot;
@property(copy) void (^changed)(void);
- (instancetype)initWithEngine:(ArionKeyboardEngine *)engine;
- (BOOL)activate:(id)client error:(NSError **)error;
- (void)deactivate;
- (void)cancel;
- (void)toggleASCII;
- (BOOL)handleEvent:(NSEvent *)event;
- (BOOL)select:(NSUInteger)index revision:(NSUInteger)revision;
- (BOOL)highlight:(NSUInteger)index revision:(NSUInteger)revision;
@end
