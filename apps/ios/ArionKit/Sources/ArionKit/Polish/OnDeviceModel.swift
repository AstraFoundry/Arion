import Foundation
import FoundationModels

/// Whether the on-device model can polish right now.
public enum ModelAvailability: Equatable, Sendable {
    case available
    /// The device cannot run Apple Intelligence.
    case unsupportedDevice
    /// Apple Intelligence is off in Settings.
    case turnedOff
    /// The model is still downloading or preparing.
    case preparing
}

/// Apple's on-device language model running the shared recipe. Requests
/// never leave the device.
public struct OnDeviceModel: Sendable {
    /// The desktop bounds the final rewrite to the same 15 seconds.
    public static let timeout: Duration = .seconds(15)

    // Transforming the user's own words is the task, so the permissive
    // content-transformation guardrails apply.
    private static var model: SystemLanguageModel {
        SystemLanguageModel(useCase: .general, guardrails: .permissiveContentTransformations)
    }

    public init() {}

    public static var availability: ModelAvailability {
        switch model.availability {
        case .available: .available
        case .unavailable(.deviceNotEligible): .unsupportedDevice
        case .unavailable(.appleIntelligenceNotEnabled): .turnedOff
        case .unavailable: .preparing
        }
    }

    public enum Failure: Error, Equatable {
        case unavailable
        case timedOut
        case rejected
        case failed
    }

    /// Runs one request and returns the validated proposal.
    public func run(_ request: RewriteRequest) async throws -> String {
        guard Self.availability == .available else { throw Failure.unavailable }
        let session = LanguageModelSession(model: Self.model, instructions: request.instructions)
        let options = GenerationOptions(sampling: .greedy, maximumResponseTokens: request.maximumTokens)
        let text: String = try await withThrowingTaskGroup(of: String.self) { group in
            group.addTask {
                do {
                    return try await session.respond(to: request.prompt, options: options).content
                } catch let error as LanguageModelSession.GenerationError {
                    if case .exceededContextWindowSize = error { throw Failure.rejected }
                    if case .guardrailViolation = error { throw Failure.rejected }
                    throw Failure.failed
                }
            }
            group.addTask {
                try await Task.sleep(for: Self.timeout)
                throw Failure.timedOut
            }
            defer { group.cancelAll() }
            guard let first = try await group.next() else { throw Failure.failed }
            return first
        }
        guard RewriteValidation.accepts(text) else { throw Failure.rejected }
        return text.trimmingCharacters(in: .whitespacesAndNewlines)
    }
}
