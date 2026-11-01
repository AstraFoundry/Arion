#import <ApplicationServices/ApplicationServices.h>
#import <Foundation/Foundation.h>
#import <unistd.h>

// Exercise the production collector against owned AX attribute fixtures. These
// stubs never inspect or modify another application's windows or text.
static NSMutableDictionary<NSNumber *, NSMutableDictionary *> *attributes;
static NSMutableArray *elements;
static pid_t next_pid=100000;
static int trusted=1, secure=0, focus_checks=0, change_focus=0;
static NSNumber *key(AXUIElementRef element) {
    pid_t pid=0; AXUIElementGetPid(element,&pid); return @(pid);
}
static AXError test_attribute(AXUIElementRef element, CFStringRef name, CFTypeRef *out) {
    id value=attributes[key(element)][(__bridge NSString *)name];
    *out=value ? CFBridgingRetain(value) : NULL;
    return value ? kAXErrorSuccess : kAXErrorAttributeUnsupported;
}
static AXError test_parameter(AXUIElementRef element, CFStringRef name, CFTypeRef param, CFTypeRef *out) {
    if (!CFEqual(name,kAXStringForRangeParameterizedAttribute)) abort();
    CFRange range; *out=NULL;
    NSString *text=attributes[key(element)][@"fixtureText"];
    if (!AXValueGetValue((AXValueRef)param,kAXValueCFRangeType,&range) ||
        range.location<0 || range.length<0 || (NSUInteger)range.location>text.length ||
        (NSUInteger)range.length>text.length-(NSUInteger)range.location) return kAXErrorIllegalArgument;
    *out=CFBridgingRetain([text substringWithRange:NSMakeRange(range.location,range.length)]);
    return kAXErrorSuccess;
}
static AXError test_timeout(AXUIElementRef element, float timeout) { return kAXErrorSuccess; }
static Boolean test_trusted(void) { return trusted; }
int arion_focus_matches(void *element,int pid) { return !change_focus || ++focus_checks<2; }
int arion_secure_input(void) { return secure; }
#define AXUIElementCopyAttributeValue test_attribute
#define AXUIElementCopyParameterizedAttributeValue test_parameter
#define AXUIElementSetMessagingTimeout test_timeout
#define AXIsProcessTrusted test_trusted
#include "../accessibility.m"
#undef AXUIElementCopyAttributeValue
#undef AXUIElementCopyParameterizedAttributeValue
#undef AXUIElementSetMessagingTimeout
#undef AXIsProcessTrusted

#define CHECK(x) do { if (!(x)) { fprintf(stderr,"failed at line %d: %s\n",__LINE__,#x); return 1; } } while (0)
static id ax_value(AXValueType type,const void *value) { return CFBridgingRelease(AXValueCreate(type,value)); }
static id node(NSString *role,NSArray *children) {
    id element=CFBridgingRelease(AXUIElementCreateApplication(next_pid++));
    CGPoint p={10,10};CGSize s={100,20};
    attributes[key((__bridge AXUIElementRef)element)]=[@{
        @"AXRole":role,@"AXPosition":ax_value(kAXValueCGPointType,&p),
        @"AXSize":ax_value(kAXValueCGSizeType,&s),@"AXChildren":children ?: @[]
    } mutableCopy];
    [elements addObject:element];return element;
}
static NSMutableDictionary *attrs(id element) { return attributes[key((__bridge AXUIElementRef)element)]; }
static id label(NSString *text) { id element=node(@"AXStaticText",nil);attrs(element)[@"AXValue"]=text;return element; }
static id window(NSArray *children) {
    id root=node(@"AXWindow",children);CGSize s={800,600};attrs(root)[@"AXSize"]=ax_value(kAXValueCGSizeType,&s);return root;
}
static int read_window(id root,char **out) {
    id field=node(@"AXTextArea",nil);attrs(field)[@"AXWindow"]=root;
    double ms=0;return arion_accessibility_window((__bridge void *)field,1,1.5,out,&ms);
}
static int no_text(id root,int expected) {
    char *out=NULL;int status=read_window(root,&out);BOOL valid=status==expected && out==NULL;free(out);return valid;
}
static int text_is(id root,NSString *expected) {
    char *out=NULL;int status=read_window(root,&out);BOOL valid=status==ARION_CONTEXT_OK && out && [@(out) isEqual:expected];free(out);return valid;
}
int main(void) { @autoreleasepool {
    attributes=[NSMutableDictionary dictionary];elements=[NSMutableArray array];
    // Chromium/Electron trees can exceed the previous 32-level limit.
    id deep=label(@"visible phrase");
    for(int i=0;i<40;i++)deep=node(@"AXGroup",@[deep]);
    CHECK(text_is(window(@[deep]),@"visible phrase"));
    // Offscreen list children still consume a bounded amount of traversal.
    NSMutableArray *wide=[NSMutableArray array];
    for(int i=0;i<600;i++) { id element=label(@"offscreen secret");CGPoint p={9000,9000};attrs(element)[@"AXPosition"]=ax_value(kAXValueCGPointType,&p);[wide addObject:element]; }
    [wide addObject:label(@"visible tail")];CHECK(text_is(window(wide),@"visible tail"));
    id qt=node(@"AXStaticText",nil);attrs(qt)[@"AXTitle"]=@"static label";
    CHECK(text_is(window(@[qt]),@"static label"));
    attrs(qt)[@"AXValue"]=@"primary value";CHECK(text_is(window(@[qt]),@"primary value"));
    // Hidden, secure and offscreen branches must not leak through label support.
    id hidden=node(@"AXGroup",@[label(@"hidden secret")]);attrs(hidden)[@"AXHidden"]=@YES;
    id password=node(@"AXTextField",@[label(@"password secret")]);attrs(password)[@"AXSubrole"]=@"AXSecureTextField";
    CHECK(text_is(window(@[hidden,password,label(@"public")]),@"public"));
    id area=node(@"AXTextArea",@[label(@"child duplicate")]);
    attrs(area)[@"fixtureText"]=@"secret visible secret";attrs(area)[@"AXValue"]=@"secret visible secret";
    CFRange range={7,7};attrs(area)[@"AXVisibleCharacterRange"]=ax_value(kAXValueCFRangeType,&range);
    CHECK(text_is(window(@[area]),@"visible"));
    // A missing range must never authorize reading an entire document or its title.
    [attrs(area) removeObjectForKey:@"AXVisibleCharacterRange"];attrs(area)[@"AXChildren"]=@[];attrs(area)[@"AXTitle"]=@"not document text";
    CHECK(no_text(window(@[area]),ARION_CONTEXT_UNSUPPORTED));
    range=CFRangeMake(-1,1);attrs(area)[@"AXVisibleCharacterRange"]=ax_value(kAXValueCFRangeType,&range);
    CHECK(no_text(window(@[area]),ARION_CONTEXT_INVALID_RANGE));
    range=CFRangeMake(0,ARION_AX_MAX_TEXT+1);attrs(area)[@"AXVisibleCharacterRange"]=ax_value(kAXValueCFRangeType,&range);
    CHECK(no_text(window(@[area]),ARION_CONTEXT_TEXT_LIMIT));
    deep=label(@"too deep");for(int i=0;i<ARION_AX_MAX_DEPTH+1;i++)deep=node(@"AXGroup",@[deep]);
    CHECK(no_text(window(@[label(@"discard partial"),deep]),ARION_CONTEXT_DEPTH_LIMIT));
    wide=[NSMutableArray array];for(int i=0;i<ARION_AX_MAX_NODES;i++)[wide addObject:node(@"AXGroup",nil)];
    CHECK(no_text(window(wide),ARION_CONTEXT_NODE_LIMIT));
    NSString *large=[@"x" stringByPaddingToLength:ARION_AX_MAX_TEXT+1 withString:@"x" startingAtIndex:0];
    CHECK(no_text(window(@[label(@"discard partial"),label(large)]),ARION_CONTEXT_TEXT_LIMIT));
    id broken=window(@[label(@"visible")]);[attrs(broken) removeObjectForKey:@"AXSize"];
    CHECK(no_text(broken,ARION_CONTEXT_WINDOW_FRAME));
    CHECK(no_text(window(@[]),ARION_CONTEXT_UNSUPPORTED));
    change_focus=1;focus_checks=0;CHECK(no_text(window(@[label(@"discard changed target")]),ARION_CONTEXT_NO_WINDOW));change_focus=0;
    secure=1;CHECK(no_text(window(@[label(@"secure input")]),ARION_CONTEXT_NO_WINDOW));secure=0;
    trusted=0;CHECK(no_text(window(@[label(@"denied")]),ARION_CONTEXT_NO_PERMISSION));
    puts("shared window collector regressions passed");
    return 0;
} }
