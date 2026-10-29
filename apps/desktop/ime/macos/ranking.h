#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

// Context ranking (上下文调频): reorders the engine's leading candidates by
// how often the user confirmed each one right after the same previous word.
// The output is always a permutation of the input (RNK-02); candidates that
// cover less input than the engine's first choice never move ahead of it.
// The context is Arion's own last commit in the active client, never text
// read from the host. All calls run on the main thread.
@interface ArionContextRanker : NSObject
// records: the persisted "key -> value" rows; persist receives every change
// set (NSString to write, NSNull to erase) and runs it off the main thread.
- (instancetype)initWithRecords:(NSDictionary<NSString *, NSString *> *)records
                        persist:(void (^)(NSDictionary<NSString *, id> *changes))persist;
// Indices into candidates, best first.
- (NSArray<NSNumber *> *)order:(NSArray<NSString *> *)candidates eligible:(const unsigned char *)eligible after:(nullable NSString *)previous;
// Records a confirmed commit, without trailing punctuation, after previous.
// No-op without a usable context.
- (void)learn:(NSString *)commit after:(nullable NSString *)previous;
// The part of a commit that serves as context: its last two characters, with
// trailing punctuation and spaces removed. Nil when nothing remains.
+ (nullable NSString *)contextOf:(nullable NSString *)commit;
@property(readonly) NSUInteger count;
@end

NS_ASSUME_NONNULL_END
