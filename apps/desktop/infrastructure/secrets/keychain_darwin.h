#ifndef ARION_KEYCHAIN_H
#define ARION_KEYCHAIN_H
#define ARION_KEYCHAIN_SIZE 1
int arion_keychain_read(const char *service, const char *account, unsigned char *out, int capacity);
int arion_keychain_write(const char *service, const char *account, const unsigned char *bytes, int length);
#endif
