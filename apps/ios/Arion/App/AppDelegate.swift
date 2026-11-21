import ArionKit
import UIKit

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    func application(_ application: UIApplication, configurationForConnecting connectingSceneSession: UISceneSession, options: UIScene.ConnectionOptions) -> UISceneConfiguration {
        UISceneConfiguration(name: nil, sessionRole: connectingSceneSession.role)
    }
}

final class SceneDelegate: UIResponder, UIWindowSceneDelegate {
    var window: UIWindow?
    private let preferences = Preferences()

    func scene(_ scene: UIScene, willConnectTo session: UISceneSession, options connectionOptions: UIScene.ConnectionOptions) {
        guard let scene = scene as? UIWindowScene else { return }
        let window = UIWindow(windowScene: scene)
        // Sunbeam checks and switches in ink; yellow is reserved for the one primary action.
        window.tintColor = Palette.foreground
        UISwitch.appearance().onTintColor = Palette.foreground
        #if DEBUG
        let arguments = ProcessInfo.processInfo.arguments
        if let index = arguments.firstIndex(of: "-keyboardPreview"), index + 1 < arguments.count {
            window.rootViewController = KeyboardPreviewController(state: arguments[index + 1])
            window.makeKeyAndVisible()
            self.window = window
            return
        }
        #endif
        let home = UINavigationController(rootViewController: HomeViewController())
        home.navigationBar.prefersLargeTitles = true
        // The large title shares the onboarding headline's serif.
        home.navigationBar.largeTitleTextAttributes = [.font: TextStyle.headline.font, .foregroundColor: Palette.foreground]
        window.rootViewController = home
        window.makeKeyAndVisible()
        self.window = window
        if !preferences.onboardingCompleted || ProcessInfo.processInfo.arguments.contains("-onboarding") {
            let onboarding = OnboardingViewController()
            onboarding.modalPresentationStyle = .fullScreen
            home.present(onboarding, animated: false)
        }
    }
}
