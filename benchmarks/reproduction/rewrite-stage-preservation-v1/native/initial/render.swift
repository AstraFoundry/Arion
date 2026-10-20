import AppKit
import Foundation
let base = URL(fileURLWithPath: CommandLine.arguments[1])
let data = try Data(contentsOf: base.appendingPathComponent("corpus.json"))
let corpus = try JSONSerialization.jsonObject(with: data) as! [String: Any]
try FileManager.default.createDirectory(at: base.appendingPathComponent("images"), withIntermediateDirectories: false)
for sample in corpus["samples"] as! [[String: Any]] {
    let id = sample["id"] as! String
    let text = sample["screenText"] as! String
    let image = NSImage(size: NSSize(width: 1400, height: 700))
    image.lockFocus()
    NSColor.white.setFill()
    NSBezierPath(rect: NSRect(x: 0, y: 0, width: 1400, height: 700)).fill()
    (text as NSString).draw(in: NSRect(x: 70, y: 100, width: 1250, height: 500), withAttributes: [.font: NSFont.systemFont(ofSize: 38), .foregroundColor: NSColor.black])
    image.unlockFocus()
    let bitmap = NSBitmapImageRep(data: image.tiffRepresentation!)!
    try bitmap.representation(using: .png, properties: [:])!.write(to: base.appendingPathComponent("images/\(id).png"), options: .withoutOverwriting)
}
