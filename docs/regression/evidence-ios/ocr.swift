// OCR tiếng Việt bằng Vision (macOS) — tesseract trên máy không có gói vie.
// swiftc -O ocr.swift -o ocr; ./ocr <ảnh.png>
import Foundation
import Vision
import AppKit
let p = CommandLine.arguments[1]
guard let ns = NSImage(contentsOfFile: p), let img = ns.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
    FileHandle.standardError.write("không đọc được ảnh\n".data(using: .utf8)!); exit(2)
}
let req = VNRecognizeTextRequest { r, _ in
    for o in (r.results as? [VNRecognizedTextObservation] ?? []) {
        if let s = o.topCandidates(1).first { print(s.string) }
    }
}
req.recognitionLevel = .accurate
req.recognitionLanguages = ["vi-VN", "en-US"]
req.usesLanguageCorrection = true
try! VNImageRequestHandler(cgImage: img).perform([req])
