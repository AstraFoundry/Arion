import Foundation
import Carbon
func v(_ s:TISInputSource,_ k:CFString)->Any?{guard let p=TISGetInputSourceProperty(s,k)else{return nil};return Unmanaged<AnyObject>.fromOpaque(p).takeUnretainedValue()}
let bundle=URL(fileURLWithPath:NSHomeDirectory()+"/Library/Input Methods/ArionInput.app")
print("register",TISRegisterInputSource(bundle as CFURL))
let filter=[kTISPropertyBundleID:"si.astrafoundry.inputmethod.arion"] as CFDictionary
let items=TISCreateInputSourceList(filter,true).takeRetainedValue() as! [TISInputSource]
for s in items{print(["id":v(s,kTISPropertyInputSourceID) ?? "","type":v(s,kTISPropertyInputSourceType) ?? "","enableCapable":v(s,kTISPropertyInputSourceIsEnableCapable) ?? false,"enabled":v(s,kTISPropertyInputSourceIsEnabled) ?? false,"selected":v(s,kTISPropertyInputSourceIsSelected) ?? false])}
for s in items where (v(s,kTISPropertyInputSourceID) as? String)=="si.astrafoundry.inputmethod.arion" {
 print("enableParent",TISEnableInputSource(s),v(s,kTISPropertyInputSourceIsEnabled) ?? "nil")
}
RunLoop.current.run(until:Date(timeIntervalSinceNow:0.3))
for s in items{print("after",v(s,kTISPropertyInputSourceID) ?? "",v(s,kTISPropertyInputSourceIsEnabled) ?? "nil")}
