import AppKit
import Carbon
import CoreServices

let parentID = "si.astrafoundry.inputmethod.arion.validation"
let bundle = URL(fileURLWithPath: NSHomeDirectory() + "/Library/Input Methods/ArionGUIValidation.app")
let output = URL(fileURLWithPath: CommandLine.arguments[1])
func property(_ source: TISInputSource, _ key: CFString) -> Any? {
    guard let pointer = TISGetInputSourceProperty(source, key) else { return nil }
    return Unmanaged<AnyObject>.fromOpaque(pointer).takeUnretainedValue()
}
func sources() -> [TISInputSource] {
    TISCreateInputSourceList([kTISPropertyBundleID: parentID] as CFDictionary, true).takeRetainedValue() as! [TISInputSource]
}
func states() -> [[String: Any]] {
    sources().map { ["id": property($0, kTISPropertyInputSourceID) ?? "", "enabled": property($0, kTISPropertyInputSourceIsEnabled) ?? false, "selected": property($0, kTISPropertyInputSourceIsSelected) ?? false] }
}
class Delegate: NSObject, NSApplicationDelegate {
    var window: NSWindow!
    var label: NSTextField!
    func applicationDidFinishLaunching(_ notification: Notification) {
        window = NSWindow(contentRect: NSRect(x: 200, y: 200, width: 600, height: 220), styleMask: [.titled, .closable], backing: .buffered, defer: false)
        window.title = "Arion input-source GUI control"
        let button = NSButton(title: "Register and enable test source", target: self, action: #selector(enable))
        button.frame = NSRect(x: 30, y: 140, width: 400, height: 40)
        window.contentView!.addSubview(button)
        label = NSTextField(wrappingLabelWithString: "Only the owned validation bundle will be registered. The external controller restores it.")
        label.frame = NSRect(x: 30, y: 30, width: 540, height: 90)
        window.contentView!.addSubview(label)
        window.center()
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }
    @objc func enable(_ sender: NSButton) {
        sender.isEnabled = false
        let before = states()
        guard !before.contains(where: { ($0["enabled"] as? Bool) == true && ($0["id"] as? String) == parentID || ($0["selected"] as? Bool) == true }), FileManager.default.fileExists(atPath: bundle.path) else { label.stringValue = "Guard refused action"; return }
        var statuses: [String: Int32] = ["launchServices": LSRegisterURL(bundle as CFURL, true), "register": TISRegisterInputSource(bundle as CFURL)]
        let registered = sources()
        for identifier in [parentID, parentID + ".pinyin"] {
            guard let source = registered.first(where: { property($0, kTISPropertyInputSourceID) as? String == identifier }) else { statuses[identifier] = -1; continue }
            statuses[identifier] = TISEnableInputSource(source)
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 1) {
            let result: [String: Any] = ["caller": "NSApplication button action on main thread", "mainThread": Thread.isMainThread, "before": before, "statuses": statuses, "after": states()]
            do { try JSONSerialization.data(withJSONObject: result, options: [.prettyPrinted, .sortedKeys]).write(to: output, options: .atomic); self.label.stringValue = "Recorded actual registration result. External controller will restore the original state." }
            catch { self.label.stringValue = "Could not save result: \(error)" }
        }
    }
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { true }
}
let app = NSApplication.shared
let delegate = Delegate()
app.delegate = delegate
app.setActivationPolicy(.regular)
app.run()
