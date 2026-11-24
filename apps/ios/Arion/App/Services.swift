import ArionKit
import AuthenticationServices
import UIKit

extension Notification.Name {
    /// Posted on the main actor whenever the account state changes.
    static let accountChanged = Notification.Name("com.astrafoundry.arion.accountChanged")
}

/// The app's long-lived services, created once on the main actor.
@MainActor
enum Services {
    static let usage = UsageLedger()
    static let cloud = CloudService()
    static let account: AccountService = {
        let service = AccountService()
        service.onChange = { _ in NotificationCenter.default.post(name: .accountChanged, object: nil) }
        return service
    }()

    /// Connects the account's sign-in sheet to the window and restores a
    /// stored account.
    static func start(window: UIWindow) {
        account.anchor = { [weak window] in window ?? ASPresentationAnchor() }
        account.restore()
        account.reportUsage()
    }
}
