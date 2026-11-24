import XCTest

/// Walks first-run setup to the settings home. Screenshots go to ARION_SCREENSHOTS when it is set
/// (TEST_RUNNER_ARION_SCREENSHOTS for xcodebuild).
@MainActor
final class FirstRunTests: XCTestCase {
    private let app = XCUIApplication()

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

    func testFirstRun() {
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

        // The keyboard's states are reviewed through the -keyboardPreview build;
        // the practice field takes focus for whichever keyboard is current.
        XCTAssertTrue(app.textViews.firstMatch.waitForExistence(timeout: 5))
        app.buttons["Done"].firstMatch.tap()
        XCTAssertTrue(app.navigationBars["Arion"].waitForExistence(timeout: 5))
        capture("home")
    }
}
