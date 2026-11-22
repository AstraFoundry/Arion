import Foundation

/// The AstraFoundry account the app signs in to, as on the desktop
/// (docs/project/account-signin.md), with a mobile public client and an app
/// redirect instead of a loopback.
public struct AccountConfig: Sendable, Equatable {
    public var issuer = URL(string: "https://astrafoundry.si")!
    public var clientID = "arion-ios"
    public var redirect = URL(string: "arion://auth/callback")!
    public var scopes = ["openid", "profile", "offline_access", "arion.sync"]
    public var resource = "https://api.arion.astrafoundry.si"

    public init() {}

    /// The account portal, `{issuer}/account`.
    public var portal: URL { issuer.appending(path: "account") }
}

public enum AccountPhase: Sendable, Equatable {
    case signedOut
    /// A stored refresh token is being exchanged at launch.
    case restoring
    case signingIn
    case signedIn
    case signingOut
}

/// The signed-in account's answer to sharing usage data, held by the platform.
public enum UsageSharing: Sendable, Equatable {
    case unknown
    case loading
    case undecided
    case shared
    case notShared
    case saving
}

/// Content-free failures, mapped to copy by the views. A cancelled sign-in is
/// not a failure and shows nothing.
public enum AccountFailure: String, Sendable, Equatable {
    case inProgress
    case denied
    case authorization
    case token
    case discovery
    case store
    case sessionEnded
    case refresh
    /// Signed out here, but the server did not confirm the revocation.
    case revokeUnconfirmed
    case usageConsent
    case generic
}

public struct AccountProfile: Sendable, Equatable {
    public var subject: String
    public var name: String
    public var email: String
    public var picture: URL?

    public init(subject: String, name: String, email: String, picture: URL?) {
        self.subject = subject
        self.name = name
        self.email = email
        self.picture = picture
    }

    /// One or two letters for the avatar when there is no picture.
    public var initials: String {
        let source = name.isEmpty ? email : name
        let words = source.split(whereSeparator: { $0 == " " || $0 == "@" || $0 == "." })
        let letters = words.prefix(2).compactMap(\.first)
        return String(letters).uppercased()
    }
}

public struct AccountState: Sendable, Equatable {
    public var phase: AccountPhase = .signedOut
    /// Kept after sign-out until another account signs in, so a restart
    /// shows the profile while restoring.
    public var profile: AccountProfile?
    public var usageSharing: UsageSharing = .unknown
    public var failure: AccountFailure?

    public init() {}

    public var signedIn: Bool { phase == .signedIn }
    public var busy: Bool { [.restoring, .signingIn, .signingOut].contains(phase) || usageSharing == .saving }
}

/// What the views call. Every method returns at once; the result arrives
/// through `state` and `onChange` on the main actor.
@MainActor
public protocol AccountServing: AnyObject {
    var state: AccountState { get }
    var onChange: ((AccountState) -> Void)? { get set }
    /// Restores a stored account at launch, without a browser.
    func restore()
    /// Opens the system web sign-in sheet. A checked usage box records
    /// consent only after a successful sign-in (ADR 044).
    func signIn(shareUsage: Bool)
    func cancel()
    /// Revokes the refresh token, then deletes it. Local data stays.
    func signOut()
    func setUsageSharing(_ share: Bool)
    /// Sends changed daily usage snapshots when signed in and sharing.
    func reportUsage()
}
