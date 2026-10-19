import Foundation
import Carbon
import Security
func value(_ source:TISInputSource,_ key:CFString)->Any? {
 guard let p = TISGetInputSourceProperty(source,key) else{return nil}
 return Unmanaged<AnyObject>.fromOpaque(p).takeUnretainedValue()
}
let args=CommandLine.arguments
let sources=TISCreateInputSourceList(nil,true).takeRetainedValue() as! [TISInputSource]
if args.count == 3 && args[1] == "select" {
 guard let source=sources.first(where:{(value($0,kTISPropertyInputSourceID) as? String)==args[2]}) else{exit(2)}
 let status=TISSelectInputSource(source)
 print(status)
 RunLoop.current.run(until:Date(timeIntervalSinceNow:0.25))
 exit(status == 0 ? 0:1)
}
let current=TISCopyCurrentKeyboardInputSource().takeRetainedValue()
let own=sources.filter{((value($0,kTISPropertyInputSourceID) as? String)?.contains("inputmethod.numbercontrol") == true && (value($0,kTISPropertyInputSourceID) as? String)?.contains("inputmethod") == true)}
var result:[String:Any] = ["current":value(current,kTISPropertyInputSourceID) ?? "", "own":own.map{ s in
 ["id":value(s,kTISPropertyInputSourceID) ?? "", "enabled":value(s,kTISPropertyInputSourceIsEnabled) ?? false,"selected":value(s,kTISPropertyInputSourceIsSelected) ?? false,"selectable":value(s,kTISPropertyInputSourceIsSelectCapable) ?? false]
}]
let query:[CFString:Any] = [kSecClass:kSecClassGenericPassword,kSecAttrService:"si.astrafoundry.arion.keyboard.learning",kSecAttrAccount:"aes256-gcm-v1"]
result["learningKeyStatus"] = SecItemCopyMatching(query as CFDictionary,nil)
print(String(data:try JSONSerialization.data(withJSONObject:result,options:[.sortedKeys]),encoding:.utf8)!)
