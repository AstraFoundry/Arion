import AppKit
import Carbon
let application=NSApplication.shared
let identifier="com.apple.inputmethod.Korean"
func value(_ source:TISInputSource,_ key:CFString)->Any? {
 guard let pointer=TISGetInputSourceProperty(source,key) else{return nil}
 return Unmanaged<AnyObject>.fromOpaque(pointer).takeUnretainedValue()
}
func sources()->[TISInputSource] {
 TISCreateInputSourceList([kTISPropertyBundleID:identifier] as CFDictionary,true).takeRetainedValue() as! [TISInputSource]
}
func state()->[[String:Any]] {
 sources().map { ["id":value($0,kTISPropertyInputSourceID) ?? "", "enabled":value($0,kTISPropertyInputSourceIsEnabled) ?? false,"selected":value($0,kTISPropertyInputSourceIsSelected) ?? false] }.sorted { ($0["id"] as! String)<($1["id"] as! String) }
}
func current()->String {
 let source=TISCopyCurrentKeyboardInputSource().takeRetainedValue()
 return value(source,kTISPropertyInputSourceID) as? String ?? ""
}
let before=state(), selected=current()
var output:[String:Any] = ["source":identifier,"before":before,"currentBefore":selected,"attempted":false]
if let parent=sources().first(where:{(value($0,kTISPropertyInputSourceID) as? String)==identifier}),
   (value(parent,kTISPropertyInputSourceIsEnabled) as? Bool)==false,
   !before.contains(where:{($0["selected"] as? Bool)==true}), selected=="com.apple.keylayout.ABC" {
 output["attempted"]=true
 output["enableStatus"]=TISEnableInputSource(parent)
 RunLoop.current.run(until:Date(timeIntervalSinceNow:0.5))
 output["afterEnable"]=state()
 output["currentAfterEnable"]=current()
 output["disableStatus"]=TISDisableInputSource(parent)
 RunLoop.current.run(until:Date(timeIntervalSinceNow:0.5))
 output["afterDisable"]=state()
 output["currentAfterDisable"]=current()
 output["restored"] = NSDictionary(dictionary:["states":before,"current":selected]).isEqual(to:["states":state(),"current":current()])
} else {
 output["reason"]="Preinstalled control is not available in the required disabled and unselected state"
 output["restored"]=true
}
print(String(data:try JSONSerialization.data(withJSONObject:output,options:[.prettyPrinted,.sortedKeys]),encoding:.utf8)!)
exit((output["restored"] as? Bool)==true ? 0 : 3)
