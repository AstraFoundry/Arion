import Foundation
import AppKit
let application = NSApplication.shared
import Carbon
func v(_ s:TISInputSource,_ k:CFString)->Any?{guard let p=TISGetInputSourceProperty(s,k)else{return nil};return Unmanaged<AnyObject>.fromOpaque(p).takeUnretainedValue()}
let sources=TISCreateInputSourceList(nil,true).takeRetainedValue() as! [TISInputSource]
for s in sources where ["arion", "chatterfly", "doubao"].contains(where: { (v(s,kTISPropertyInputSourceID) as? String ?? "").contains($0) }) {
 print(["id":v(s,kTISPropertyInputSourceID) ?? "", "type":v(s,kTISPropertyInputSourceType) ?? "", "enabled":v(s,kTISPropertyInputSourceIsEnabled) ?? false,"languages":v(s,kTISPropertyInputSourceLanguages) ?? [],"name":v(s,kTISPropertyLocalizedName) ?? ""])
}
