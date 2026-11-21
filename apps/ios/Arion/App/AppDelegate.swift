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
        let home = UINavigationController(rootViewController: HomeViewController())
        home.navigationBar.prefersLargeTitles = true
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
