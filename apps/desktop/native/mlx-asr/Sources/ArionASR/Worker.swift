import Foundation
import Darwin
import MLX
import MLXAudioSTT
import MLXRandom

struct Request: Decodable, Sendable {
    let type: String
    let id: String
    var seq: Int?
    var pcm: String?
    var sample_rate: Int?
}

struct Response: Encodable {
    let type: String
    var id: String?
    var text: String?
    var stop: String?
    var `protocol`: Int?
    var version: String?
}

// The actor owns the resident model and serializes inference. Cancellation
// remains available while asynchronous token generation is running.
actor Worker {
    private var model: Qwen3ASRModel?
    private var active: Task<Void, Never>?
    private var activeID: String?
    private var chunks: [String: Data] = [:]
    private var sequences: [String: Int] = [:]

    func submit(_ request: Request) {
        if request.type == "cancel" {
            chunks.removeValue(forKey: request.id)
            sequences.removeValue(forKey: request.id)
            if activeID == request.id { active?.cancel() }
            return
        }
        if request.type == "chunk" {
            guard request.seq == sequences[request.id, default: 0],
                  let pcm = request.pcm, let data = Data(base64Encoded: pcm), data.count % 2 == 0,
                  chunks[request.id, default: Data()].count + data.count <= 16000 * 25 * 2 else {
                reply(Response(type: "error", id: request.id))
                return
            }
            chunks[request.id, default: Data()].append(data)
            sequences[request.id, default: 0] += 1
            return
        }
        guard request.type == "transcribe", active == nil else {
            reply(Response(type: "error", id: request.id))
            return
        }
        activeID = request.id
        active = Task { await execute(request) }
    }

    private func execute(_ request: Request) async {
        do {
            guard let model, request.sample_rate == 16000,
                  let data = chunks.removeValue(forKey: request.id), !data.isEmpty else {
                throw WorkerError.invalidRequest
            }
            sequences.removeValue(forKey: request.id)
                let samples: [Float] = data.withUnsafeBytes { bytes in
                    stride(from: 0, to: bytes.count, by: 2).map {
                        Float(Int16(littleEndian: bytes.loadUnaligned(fromByteOffset: $0, as: Int16.self))) / 32768
                    }
                }
                guard samples.allSatisfy(\.isFinite) else { throw WorkerError.invalidRequest }
                MLXRandom.seed(11)
                var output: String?
                for try await event in model.generateStream(audio: MLXArray(samples), maxTokens: 128, temperature: 0) {
                    try Task.checkCancellation()
                    if case .result(let result) = event {
                        guard result.generationTokens < 128 else { throw WorkerError.tokenLimit }
                        output = result.text
                    }
                }
                try Task.checkCancellation()
                guard let output else { throw WorkerError.noResult }
            reply(Response(type: "end", id: request.id, text: output, stop: "complete"))
        } catch is CancellationError {
            reply(Response(type: "cancelled", id: request.id))
        } catch {
            reply(Response(type: "error", id: request.id))
        }
        active = nil
        activeID = nil
    }

    func load(_ path: String) async throws {
        // Only verified catalog files enter the loader. Derived tokenizer.json
        // is generated in a private temporary directory, never trusted from cache.
        let source = URL(fileURLWithPath: path, isDirectory: true)
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent("arion-asr-" + UUID().uuidString)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
        defer { try? FileManager.default.removeItem(at: directory) }
        for name in ["config.json", "preprocessor_config.json", "tokenizer_config.json", "merges.txt", "vocab.json", "generation_config.json", "model.safetensors", "model.safetensors.index.json"] {
            try FileManager.default.createSymbolicLink(at: directory.appendingPathComponent(name), withDestinationURL: source.appendingPathComponent(name))
        }
        model = try await Qwen3ASRModel.fromModelDirectory(directory)
        reply(Response(type: "ready", protocol: 3, version: "arion-mlx-1"))
    }

    func shutdown() async {
        active?.cancel()
        await active?.value
        model = nil
    }
}

enum WorkerError: Error { case invalidRequest, noResult, tokenLimit }

// Preserve a dedicated protocol descriptor before silencing upstream print calls.
let protocolOutput = FileHandle(fileDescriptor: dup(STDOUT_FILENO), closeOnDealloc: true)

func reply(_ response: Response) {
    guard let data = try? JSONEncoder().encode(response) else { return }
    protocolOutput.write(data + Data([10]))
}

@main
struct ArionASR {
    static func main() async {
        _ = protocolOutput
        let sink = open("/dev/null", O_WRONLY)
        guard sink >= 0, dup2(sink, STDOUT_FILENO) >= 0 else { return }
        close(sink)
        let worker = Worker()
        guard CommandLine.arguments.count == 3, CommandLine.arguments[1] == "worker" else { return }
        do { try await worker.load(CommandLine.arguments[2]) }
        catch { reply(Response(type: "fatal")); return }
        await Task.detached {
            while let line = readLine(strippingNewline: true) {
                guard line.utf8.count <= 3 * 1024 * 1024,
                      let request = try? JSONDecoder().decode(Request.self, from: Data(line.utf8)) else { break }
                await worker.submit(request)
            }
            await worker.shutdown()
        }.value
    }
}
