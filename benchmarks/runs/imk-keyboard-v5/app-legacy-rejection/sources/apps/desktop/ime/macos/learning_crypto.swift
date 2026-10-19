import CryptoKit
import Foundation

// The C boundary copies its result into malloc-owned memory, released by the
// Objective-C++ caller. CryptoKit owns nonce generation and GCM verification.
private func transfer(_ data: Data,
                      output: UnsafeMutablePointer<UnsafeMutableRawPointer?>,
                      outputCount: UnsafeMutablePointer<Int>) -> Int32 {
    guard let allocation = malloc(max(data.count, 1)) else { return 0 }
    data.copyBytes(to: allocation.assumingMemoryBound(to: UInt8.self), count: data.count)
    output.pointee = allocation
    outputCount.pointee = data.count
    return 1
}

@_cdecl("arion_learning_seal")
public func arionLearningSeal(_ input: UnsafeRawPointer, _ inputCount: Int,
                             _ key: UnsafeRawPointer, _ keyCount: Int,
                             _ aad: UnsafeRawPointer, _ aadCount: Int,
                             _ output: UnsafeMutablePointer<UnsafeMutableRawPointer?>,
                             _ outputCount: UnsafeMutablePointer<Int>) -> Int32 {
    guard keyCount == 32, inputCount >= 0, aadCount > 0 else { return 0 }
    do {
        let sealed = try AES.GCM.seal(Data(bytes: input, count: inputCount),
                                     using: SymmetricKey(data: Data(bytes: key, count: keyCount)),
                                     authenticating: Data(bytes: aad, count: aadCount))
        guard let combined = sealed.combined else { return 0 }
        return transfer(combined, output: output, outputCount: outputCount)
    } catch {
        return 0
    }
}

@_cdecl("arion_learning_open")
public func arionLearningOpen(_ input: UnsafeRawPointer, _ inputCount: Int,
                             _ key: UnsafeRawPointer, _ keyCount: Int,
                             _ aad: UnsafeRawPointer, _ aadCount: Int,
                             _ output: UnsafeMutablePointer<UnsafeMutableRawPointer?>,
                             _ outputCount: UnsafeMutablePointer<Int>) -> Int32 {
    guard keyCount == 32, inputCount >= 28, aadCount > 0 else { return 0 }
    do {
        let sealed = try AES.GCM.SealedBox(combined: Data(bytes: input, count: inputCount))
        let plaintext = try AES.GCM.open(sealed,
                                        using: SymmetricKey(data: Data(bytes: key, count: keyCount)),
                                        authenticating: Data(bytes: aad, count: aadCount))
        return transfer(plaintext, output: output, outputCount: outputCount)
    } catch {
        return 0
    }
}

// Public CryptoKit hashing for immutable keyboard-data verification at startup.
@_cdecl("arion_keyboard_sha256")
public func arionKeyboardSHA256(_ input: UnsafeRawPointer?, _ length: Int,
                               _ output: UnsafeMutablePointer<UInt8>?) -> Int32 {
    guard length >= 0, let output, length == 0 || input != nil else { return 0 }
    let data = length == 0 ? Data() : Data(bytes: input!, count: length)
    let digest = SHA256.hash(data: data)
    for (index, byte) in digest.enumerated() { output[index] = byte }
    return 1
}
