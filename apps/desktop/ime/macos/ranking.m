#import "ranking.h"
#include "../shared/ranking_bridge.h"
#include <stdlib.h>

@interface ArionContextRanker ()
@property arion_ranker *core;
@property(copy) void (^persist)(NSDictionary<NSString *,id> *changes);
@end

static void collectChange(const char *key,const char *value,void *context) {
    NSMutableDictionary *changes=(__bridge NSMutableDictionary *)context;
    NSString *name=[NSString stringWithUTF8String:key];
    if (value) changes[name]=[NSString stringWithUTF8String:value];
    else changes[name]=NSNull.null;
}

@implementation ArionContextRanker
- (instancetype)initWithRecords:(NSDictionary<NSString *,NSString *> *)records
                        persist:(void (^)(NSDictionary<NSString *,id> *))persist {
    if (!(self=[super init])) return nil;
    NSMutableData *keys=[NSMutableData dataWithLength:records.count*sizeof(const char *)];
    NSMutableData *values=[NSMutableData dataWithLength:records.count*sizeof(const char *)];
    const char **k=keys.mutableBytes,**v=values.mutableBytes;
    NSUInteger i=0;
    for (NSString *key in records) {k[i]=key.UTF8String;v[i]=records[key].UTF8String;i++;}
    self.core=arion_ranker_create(k,v,records.count);
    self.persist=persist;
    return self;
}
- (void)dealloc {arion_ranker_destroy(self.core);}
- (NSUInteger)count {return arion_ranker_count(self.core);}
+ (NSString *)contextOf:(NSString *)commit {
    char *context=arion_ranker_context(commit.UTF8String);
    if (!context) return nil;
    NSString *result=[NSString stringWithUTF8String:context];free(context);return result;
}
- (NSArray<NSNumber *> *)order:(NSArray<NSString *> *)candidates after:(NSString *)previous {
    NSMutableData *values=[NSMutableData dataWithLength:candidates.count*sizeof(const char *)];
    NSMutableData *indices=[NSMutableData dataWithLength:candidates.count*sizeof(size_t)];
    const char **v=values.mutableBytes;size_t *order=indices.mutableBytes;
    for (NSUInteger i=0;i<candidates.count;i++) v[i]=candidates[i].UTF8String;
    arion_ranker_order(self.core,v,candidates.count,previous.UTF8String,order);
    NSMutableArray<NSNumber *> *result=[NSMutableArray arrayWithCapacity:candidates.count];
    for (NSUInteger i=0;i<candidates.count;i++) [result addObject:@(order[i])];
    return result;
}
- (void)learn:(NSString *)commit after:(NSString *)previous {
    NSMutableDictionary *changes=[NSMutableDictionary dictionary];
    arion_ranker_learn(self.core,commit.UTF8String,previous.UTF8String,collectChange,(__bridge void *)changes);
    if (changes.count && self.persist) self.persist(changes);
}
@end
