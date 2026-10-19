import Foundation
import AppKit
import CoreServices
let application = NSApplication.shared
import Carbon
func v(_ s:TISInputSource,_ k:CFString)->Any?{guard let p=TISGetInputSourceProperty(s,k)else{return nil};return Unmanaged<AnyObject>.fromOpaque(p).takeUnretainedValue()}
let bundle=URL(fileURLWithPath:NSHomeDirectory()+"/Library/Input Methods/ArionInput.app")
print("launchServices",LSRegisterURL(bundle as CFURL,true))
print("register",TISRegisterInputSource(bundle as CFURL))
let filter=[kTISPropertyBundleID:"si.astrafoundry.inputmethod.arion"] as CFDictionary
let items=TISCreateInputSourceList(filter,true).takeRetainedValue() as! [TISInputSource]
for s in items{print(["id":v(s,kTISPropertyInputSourceID) ?? "","type":v(s,kTISPropertyInputSourceType) ?? "","enableCapable":v(s,kTISPropertyInputSourceIsEnableCapable) ?? false,"enabled":v(s,kTISPropertyInputSourceIsEnabled) ?? false,"selected":v(s,kTISPropertyInputSourceIsSelected) ?? false,"languages":v(s,kTISPropertyInputSourceLanguages) ?? [],"name":v(s,kTISPropertyLocalizedName) ?? ""])}
for id in ["si.astrafoundry.inputmethod.arion.pinyin", "si.astrafoundry.inputmethod.arion"] {
 for source in items where (v(source,kTISPropertyInputSourceID) as? String)==id {print("disable",id,TISDisableInputSource(source))}
}
for id in ["si.astrafoundry.inputmethod.arion", "si.astrafoundry.inputmethod.arion.pinyin"] {
 for source in items where (v(source,kTISPropertyInputSourceID) as? String)==id {print("enable",id,TISEnableInputSource(source))}
}
RunLoop.current.run(until:Date(timeIntervalSinceNow:1))
for source in items {print("after",v(source,kTISPropertyInputSourceID) ?? "",v(source,kTISPropertyInputSourceIsEnabled) ?? "nil")}
