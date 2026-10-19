#pragma once
#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

#ifdef __cplusplus
extern "C" {
#endif

// Registers the encrypted backend before creating any Rime sessions. The
// directory holds ciphertext only; the 256-bit key stays in the Data Protection Keychain.
BOOL ArionRegisterEncryptedLearning(NSString *directory, NSError **error);
// Explicitly deletes learned snapshots after all Rime sessions are closed and
// the engine is finalized. Refuses active file locks; never changes the key.
BOOL ArionResetEncryptedLearning(NSString *directory, NSError **error);
// Returns a content-free failure if loading or persisting learning failed.
NSError * _Nullable ArionEncryptedLearningError(void);

#ifdef __cplusplus
}
namespace rime { class Db; }
#ifdef ARION_LEARNING_TESTING
// Synthetic-key seam for source tests; never reads or writes the user's Keychain.
BOOL ArionLearningTestVerifiedKeyPolicy(NSDictionary *attributes);
BOOL ArionLearningTestMayCreateKey(NSArray<NSString *> *files);
BOOL ArionRegisterTestEncryptedLearning(NSString *directory, NSData *key, NSError **error);
rime::Db * _Nullable ArionLearningTestDatabase(NSString *file, NSString *name,
                                              NSData * _Nullable key);
#endif
#endif

NS_ASSUME_NONNULL_END
