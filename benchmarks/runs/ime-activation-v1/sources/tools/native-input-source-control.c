#include <Carbon/Carbon.h>
#include <stdio.h>
int main(int argc, char **argv) {
 if (argc == 2) {
  CFStringRef wanted = CFStringCreateWithCString(NULL, argv[1], kCFStringEncodingUTF8);
  const void *keys[] = {kTISPropertyInputSourceID};
  const void *values[] = {wanted};
  CFDictionaryRef filter = CFDictionaryCreate(NULL, keys, values, 1, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
  CFArrayRef sources = TISCreateInputSourceList(filter, false);
  OSStatus status = sources && CFArrayGetCount(sources) ? TISSelectInputSource((TISInputSourceRef)CFArrayGetValueAtIndex(sources, 0)) : paramErr;
  if (sources) CFRelease(sources);
  CFRelease(filter);CFRelease(wanted);
  if (status != noErr) return 1;
 }
 TISInputSourceRef current = TISCopyCurrentKeyboardInputSource();
 if (!current) return 2;
 CFStringRef identifier = TISGetInputSourceProperty(current, kTISPropertyInputSourceID);
 char value[1024];
 if (!identifier || !CFStringGetCString(identifier, value, sizeof(value), kCFStringEncodingUTF8)) {CFRelease(current);return 3;}
 puts(value);CFRelease(current);return 0;
}
