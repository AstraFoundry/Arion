import AppKit
import CoreServices
import Carbon
let application = NSApplication.shared
let args = CommandLine.arguments
let bundle = URL(fileURLWithPath:args[1])
let operation = args[2]
let parentID = args[3]
let modeID = parentID + ".pinyin"
func value(_ source:TISInputSource,_ key:CFString)->Any? {
 guard let pointer=TISGetInputSourceProperty(source,key) else{return nil}
 return Unmanaged<AnyObject>.fromOpaque(pointer).takeUnretainedValue()
}
var statuses:[String:Int32]=[:]
if operation == "enable" {
 statuses["launchServices"] = LSRegisterURL(bundle as CFURL,true)
 statuses["register"] = TISRegisterInputSource(bundle as CFURL)
}
func matching()->[TISInputSource] {
 let filter=[kTISPropertyBundleID:parentID] as CFDictionary
 return TISCreateInputSourceList(filter,true).takeRetainedValue() as! [TISInputSource]
}
let sources=matching()
for identifier in operation == "disable" ? [modeID,parentID] : [parentID,modeID] {
 guard let source=sources.first(where:{(value($0,kTISPropertyInputSourceID) as? String)==identifier}) else{statuses[identifier] = -1;continue}
 statuses[identifier] = operation == "disable" ? TISDisableInputSource(source) : TISEnableInputSource(source)
}
RunLoop.current.run(until:Date(timeIntervalSinceNow:0.5))
let states = matching().map { source -> [String:Any] in
 ["id":value(source,kTISPropertyInputSourceID) ?? "", "enabled":value(source,kTISPropertyInputSourceIsEnabled) ?? false, "selected":value(source,kTISPropertyInputSourceIsSelected) ?? false]
}
let output:[String:Any] = ["operation":operation,"statuses":statuses,"sources":states]
print(String(data:try JSONSerialization.data(withJSONObject:output,options:[.sortedKeys]),encoding:.utf8)!)
