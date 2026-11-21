import XCTest

/// Walks first-run setup into the practice field and records the keyboard's
/// states. Screenshots go to ARION_SCREENSHOTS when it is set
/// (TEST_RUNNER_ARION_SCREENSHOTS for xcodebuild).
@MainActor
final class KeyboardFlowTests: XCTestCase {
    private let app = XCUIApplication()
    private let settings = XCUIApplication(bundleIdentifier: "com.apple.Preferences")

    override func setUp() async throws {
        continueAfterFailure = true
    }

    private func capture(_ name: String) {
        let shot = XCUIScreen.main.screenshot()
        let attachment = XCTAttachment(screenshot: shot)
        attachment.name = name
        attachment.lifetime = .keepAlways
        add(attachment)
        if let directory = ProcessInfo.processInfo.environment["ARION_SCREENSHOTS"] {
            try? shot.pngRepresentation.write(to: URL(fileURLWithPath: directory).appendingPathComponent("\(name).png"))
        }
    }

    /// Adds the Arion keyboard and turns on Allow Full Access in Settings.
    private func enableKeyboard() {
        settings.terminate()
        settings.launch()
        func open(_ label: String) -> Bool {
            // A page's title repeats the row that opened it; the row is the last match.
            let matches = settings.descendants(matching: .any).matching(NSPredicate(format: "label == %@ AND (elementType == %d OR elementType == %d)", label, XCUIElement.ElementType.staticText.rawValue, XCUIElement.ElementType.button.rawValue))
            _ = matches.firstMatch.waitForExistence(timeout: 5)
            let item = matches.count > 1 ? matches.element(boundBy: matches.count - 1) : matches.firstMatch
            for _ in 0..<4 where !item.isHittable { settings.swipeUp() }
            guard item.waitForExistence(timeout: 5) else {
                capture("settings-missing-\(label)")
                return false
            }
            item.tap()
            return true
        }
        guard open("General"), open("Keyboard"), open("Keyboards") else { return }
        if !settings.staticTexts["Arion"].waitForExistence(timeout: 2) {
            guard open("Add New Keyboard") else { return }
            capture("settings-add-keyboard")
            let search = settings.searchFields.firstMatch
            if search.waitForExistence(timeout: 3) {
                search.tap()
                search.typeText("Arion")
            }
            guard open("Arion") else { return }
        }
        guard open("Arion") else { return }
        let toggle = settings.switches.firstMatch
        guard toggle.waitForExistence(timeout: 5) else { return }
        if (toggle.value as? String) != "1" {
            toggle.tap()
            let allow = settings.alerts.buttons["Allow"]
            if allow.waitForExistence(timeout: 3) { allow.tap() }
        }
        capture("settings-full-access")
    }

    func testFirstRunAndKeyboard() {
        // iOS lists an app's keyboard once the app has run.
        app.launch()
        app.terminate()
        enableKeyboard()
        addUIInterruptionMonitor(withDescription: "Permissions") { alert in
            for label in ["Allow", "OK", "允许", "好"] where alert.buttons[label].exists {
                alert.buttons[label].tap()
                return true
            }
            alert.buttons.element(boundBy: alert.buttons.count - 1).tap()
            return true
        }
        app.launchArguments = ["-onboarding"]
        app.launch()
        capture("onboarding-welcome")
        app.buttons["Continue"].tap()
        capture("onboarding-microphone")
        app.buttons["Allow"].tap()
        sleep(2)
        // Interacting lets the interruption monitor answer the system prompts.
        app.swipeUp()
        if app.buttons["Open Settings"].waitForExistence(timeout: 5) {
            capture("onboarding-keyboard")
            // Continue is the plain secondary action on this step.
            app.buttons["Continue"].tap()
            sleep(2)
        }
        capture("onboarding-practice")

        app.textViews.firstMatch.tap()
        let keyboard = app.keyboards.firstMatch
        _ = keyboard.waitForExistence(timeout: 5)
        let dictate = app.buttons.matching(NSPredicate(format: "label IN %@", ["Dictate", "听写"])).firstMatch
        if dictate.waitForExistence(timeout: 5) {
            capture("keyboard-idle")
            dictate.tap()
            sleep(1)
            capture("keyboard-listening")
            sleep(4)
            capture("keyboard-after")
        } else {
            capture("keyboard-missing")
        }
    }
}
