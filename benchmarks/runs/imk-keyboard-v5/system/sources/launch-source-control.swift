import AppKit
import CoreServices
let arguments = CommandLine.arguments
let bundle = URL(fileURLWithPath: arguments[1])
let status = LSRegisterURL(bundle as CFURL, true)
guard status == noErr else { print("{\"error\":\"launch-services\"}"); exit(1) }
let configuration = NSWorkspace.OpenConfiguration()
configuration.activates = false
configuration.createsNewApplicationInstance = true
configuration.arguments = [arguments[2]]
var finished = false
var result: [String:Any] = [:]
NSWorkspace.shared.openApplication(at:bundle, configuration:configuration) { application,error in
 if let application { result = ["pid":application.processIdentifier, "bundleIdentifier":application.bundleIdentifier ?? ""] }
 else { result = ["error":"launch-failed", "code":(error as NSError?)?.code ?? -1] }
 finished = true
}
while !finished { RunLoop.current.run(until:Date(timeIntervalSinceNow:0.05)) }
print(String(data:try JSONSerialization.data(withJSONObject:result,options:[.sortedKeys]),encoding:.utf8)!)
exit(result["error"] == nil ? 0 : 1)
