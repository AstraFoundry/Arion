#pragma once
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <bsm/libbsm.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <unistd.h>
#include <stdlib.h>
#include <string.h>

static SecCodeRef arion_socket_peer(int fd, uint32_t *pid, uint32_t *uid) {
    audit_token_t token;
    socklen_t size = sizeof(token);
    if (getsockopt(fd, SOL_LOCAL, LOCAL_PEERTOKEN, &token, &size) != 0 || size != sizeof(token) ||
        audit_token_to_euid(token) != geteuid()) return NULL;
    CFDataRef data = CFDataCreate(NULL, (const UInt8 *)&token, sizeof(token));
    if (!data) return NULL;
    const void *keys[] = {kSecGuestAttributeAudit};
    const void *values[] = {data};
    CFDictionaryRef attributes = CFDictionaryCreate(NULL, keys, values, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    SecCodeRef code = NULL;
    OSStatus status = attributes ? SecCodeCopyGuestWithAttributes(NULL, attributes, kSecCSDefaultFlags, &code) : errSecAllocate;
    if (attributes) CFRelease(attributes);
    CFRelease(data);
    if (status != errSecSuccess) return NULL;
    *pid = (uint32_t)audit_token_to_pid(token);
    *uid = (uint32_t)audit_token_to_euid(token);
    return code;
}

static int arion_peer_valid(SecCodeRef code, const char *rule) {
    SecRequirementRef requirement = NULL;
    if (rule) {
        CFStringRef text = CFStringCreateWithCString(NULL, rule, kCFStringEncodingUTF8);
        if (!text) return 0;
        OSStatus parsed = SecRequirementCreateWithString(text, kSecCSDefaultFlags, &requirement);
        CFRelease(text);
        if (parsed != errSecSuccess) return 0;
    }
    OSStatus status = SecCodeCheckValidity(code, kSecCSStrictValidate, requirement);
    if (requirement) CFRelease(requirement);
    return status == errSecSuccess;
}

// The trusted publisher comes from this running, Developer ID signed binary.
// No peer-supplied field, environment setting or unsigned metadata chooses it.
static SecRequirementRef arion_peer_publisher_requirement(const char *identifier) {
    if (!identifier || !*identifier || strlen(identifier)>255) return NULL;
    for (const char *p=identifier; *p; ++p)
        if (!((*p>='a' && *p<='z') || (*p>='A' && *p<='Z') || (*p>='0' && *p<='9') || *p=='.' || *p=='-')) return NULL;
    const char *developer="anchor apple generic and certificate 1[field.1.2.840.113635.100.6.2.6] exists and certificate leaf[field.1.2.840.113635.100.6.1.13] exists";
    SecCodeRef own=NULL;
    if (SecCodeCopySelf(kSecCSDefaultFlags,&own)!=errSecSuccess) return NULL;
    if (!arion_peer_valid(own,developer)) { CFRelease(own); return NULL; }
    CFDictionaryRef info=NULL;
    OSStatus status=SecCodeCopySigningInformation(own,kSecCSSigningInformation,&info);
    CFRelease(own);
    if (status!=errSecSuccess || !info) return NULL;
    CFStringRef team=(CFStringRef)CFDictionaryGetValue(info,kSecCodeInfoTeamIdentifier);
    char teamID[11]={0};
    int valid=team && CFGetTypeID(team)==CFStringGetTypeID() && CFStringGetLength(team)==10 &&
        CFStringGetCString(team,teamID,sizeof(teamID),kCFStringEncodingASCII);
    for (size_t i=0; valid && i<10; ++i)
        valid=(teamID[i]>='A' && teamID[i]<='Z') || (teamID[i]>='0' && teamID[i]<='9');
    CFRelease(info);
    if (!valid) return NULL;
    CFStringRef rule=CFStringCreateWithFormat(NULL,NULL,CFSTR("identifier \"%s\" and %s and certificate leaf[subject.OU] = \"%s\""),identifier,developer,teamID);
    if (!rule) return NULL;
    SecRequirementRef requirement=NULL;
    status=SecRequirementCreateWithString(rule,kSecCSDefaultFlags,&requirement);
    CFRelease(rule);
    return status==errSecSuccess ? requirement : NULL;
}

static int arion_peer_matches(SecCodeRef code, SecRequirementRef requirement) {
    return code && requirement && SecCodeCheckValidity(code,kSecCSStrictValidate,requirement)==errSecSuccess;
}
