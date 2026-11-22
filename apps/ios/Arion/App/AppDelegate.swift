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
        let tabs = UITabBarController()
        tabs.viewControllers = [
            Self.tab(OverviewViewController(), title: L10n.tabOverview, icon: .chartLine),
            Self.tab(DictationViewController(), title: L10n.tabDictation, icon: .audioLines),
            Self.tab(PolishViewController(), title: L10n.tabPolish, icon: .sparkles),
            Self.tab(VocabularyViewController(), title: L10n.tabVocabulary, icon: .bookOpen),
            Self.tab(AccountViewController(), title: L10n.tabAccount, icon: .circleUser),
        ]
        window.rootViewController = tabs
        window.makeKeyAndVisible()
        self.window = window
        Services.start(window: window)
        if !preferences.onboardingCompleted || ProcessInfo.processInfo.arguments.contains("-onboarding") {
            let onboarding = OnboardingViewController()
            onboarding.modalPresentationStyle = .fullScreen
            tabs.present(onboarding, animated: false)
        }
    }

    func sceneDidBecomeActive(_ scene: UIScene) {
        Services.account.reportUsage()
    }

    private static func tab(_ root: UIViewController, title: String, icon: Icon) -> UINavigationController {
        // A tab's root shows its large title; pages it opens use the inline one.
        root.navigationItem.largeTitleDisplayMode = .always
        let navigation = UINavigationController(rootViewController: root)
        navigation.navigationBar.prefersLargeTitles = true
        // Large titles share the onboarding headline's serif.
        navigation.navigationBar.largeTitleTextAttributes = [.font: TextStyle.headline.font, .foregroundColor: Palette.foreground]
        navigation.tabBarItem = UITabBarItem(title: title, image: icon.image(size: 24), selectedImage: nil)
        return navigation
    }
}
