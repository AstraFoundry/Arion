#import <Cocoa/Cocoa.h>

// Owns one client's provisional voice range. All calls run on the main thread.
@interface ArionComposition : NSObject
@property(readonly, copy) NSString *activation;
@property(readonly, copy) NSString *session;
@property(readonly) BOOL live;
@property(readonly) BOOL composing;
- (void)activate:(id)client;
- (void)invalidate;
- (NSDictionary *)apply:(NSDictionary *)message;
@end
