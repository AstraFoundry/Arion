import AVFAudio
import UIKit
import WebKit

/// Captures the microphone from inside the keyboard extension.
///
/// A keyboard process cannot keep AVAudioEngine running, but WebKit captures
/// media in its own processes. An invisible web view in the keyboard's window
/// opens `getUserMedia` and posts 16 kHz mono PCM back to native code. The
/// approach follows stephancill/ios-keyboard-dictation (MIT); see ADR 050.
///
/// The page stops the microphone itself when native heartbeats stop for three
/// seconds, so a dismissed or suspended keyboard cannot keep recording.
@MainActor
public final class WebMicrophone: NSObject {
    public static let sampleRate: Double = 16_000
    public static let format = AVAudioFormat(commonFormat: .pcmFormatInt16, sampleRate: sampleRate, channels: 1, interleaved: true)!

    public enum Event {
        case started
        /// A chunk of audio and its RMS level in 0...1.
        case audio(AVAudioPCMBuffer, level: Float)
        case ended(reason: String)
    }

    public var onEvent: ((Event) -> Void)?
    public private(set) var isReady = false
    public private(set) var isCapturing = false

    private let webView: WKWebView
    private var heartbeat: Timer?
    private var pendingStart = false
    private static let origin = URL(string: "https://keyboard.arion.invalid/")

    public override init() {
        let configuration = WKWebViewConfiguration()
        configuration.allowsInlineMediaPlayback = true
        configuration.mediaTypesRequiringUserActionForPlayback = []
        configuration.websiteDataStore = .nonPersistent()
        webView = WKWebView(frame: CGRect(x: 0, y: 0, width: 1, height: 1), configuration: configuration)
        super.init()
        configuration.userContentController.add(MessageRelay(self), name: "capture")
        webView.uiDelegate = self
        webView.navigationDelegate = self
        // WebKit throttles a web view outside a window, so it stays in the
        // hierarchy, nearly transparent, instead of hidden.
        webView.alpha = 0.01
        webView.isUserInteractionEnabled = false
        webView.accessibilityElementsHidden = true
        webView.isOpaque = false
    }

    /// Puts the capture page in the keyboard's view. Capture works only while attached.
    public func attach(to view: UIView) {
        guard webView.superview == nil else { return }
        view.insertSubview(webView, at: 0)
        load()
    }

    public func start() {
        guard !isCapturing else { return }
        guard isReady else {
            pendingStart = true
            return
        }
        pendingStart = false
        isCapturing = true
        heartbeat = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            MainActor.assumeIsolated { _ = self?.webView.evaluateJavaScript("captureHeartbeat(); 0") }
        }
        webView.evaluateJavaScript("startCapture(); 0") { [weak self] _, error in
            guard let error else { return }
            MainActor.assumeIsolated { self?.finish(reason: "start: \(error.localizedDescription)") }
        }
    }

    public func stop() {
        pendingStart = false
        guard isCapturing else { return }
        webView.evaluateJavaScript("stopCapture(); 0")
        finish(reason: "stopped")
    }

    public func detach() {
        stop()
        webView.stopLoading()
        webView.removeFromSuperview()
        isReady = false
    }

    private func load() {
        webView.loadHTMLString(Self.page, baseURL: Self.origin)
    }

    private func finish(reason: String) {
        heartbeat?.invalidate()
        heartbeat = nil
        guard isCapturing else { return }
        isCapturing = false
        onEvent?(.ended(reason: reason))
    }

    fileprivate func receive(_ body: Any) {
        guard let message = body as? [String: Any], let type = message["type"] as? String else { return }
        switch type {
        case "ready":
            isReady = true
            if pendingStart { start() }
        case "started":
            onEvent?(.started)
        case "audio":
            guard isCapturing,
                  let encoded = message["data"] as? String,
                  let data = Data(base64Encoded: encoded),
                  let buffer = Self.buffer(from: data) else { return }
            let level = (message["rms"] as? NSNumber)?.floatValue ?? 0
            onEvent?(.audio(buffer, level: min(1, level * 4)))
        case "interrupted", "failed":
            finish(reason: "\(type): \(message["reason"] as? String ?? "")")
        default:
            break
        }
    }

    static func buffer(from data: Data) -> AVAudioPCMBuffer? {
        let frames = AVAudioFrameCount(data.count / 2)
        guard frames > 0, let buffer = AVAudioPCMBuffer(pcmFormat: format, frameCapacity: frames) else { return nil }
        buffer.frameLength = frames
        data.withUnsafeBytes { raw in
            guard let source = raw.baseAddress, let target = buffer.int16ChannelData?[0] else { return }
            memcpy(target, source, Int(frames) * 2)
        }
        return buffer
    }

    private static let page = """
    <!doctype html><html><body><script>
    const targetRate = \(Int(sampleRate));
    const post = m => window.webkit.messageHandlers.capture.postMessage(m);
    const encode = s => {
      const bytes = new Uint8Array(s.buffer, 0, s.length * 2);
      let binary = '';
      for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode.apply(null, bytes.subarray(i, i + 0x8000));
      return btoa(binary);
    };
    let lastBeat = performance.now();
    window.captureHeartbeat = () => { lastBeat = performance.now(); };
    let stop = () => {};
    window.stopCapture = () => stop();
    window.startCapture = async () => {
      let stopping = false, stream, context, source, processor, sink, watchdog;
      const session = navigator.audioSession;
      const previous = session?.type ?? 'auto';
      const cleanup = () => {
        if (stopping) return;
        stopping = true;
        clearInterval(watchdog);
        if (processor) { processor.onaudioprocess = null; processor.disconnect(); }
        source?.disconnect(); sink?.disconnect();
        stream?.getTracks().forEach(t => t.stop());
        context?.close().catch(() => {});
        try { if (session) session.type = previous; } catch (_) {}
      };
      const interrupted = reason => { if (!stopping) { cleanup(); post({ type: 'interrupted', reason }); } };
      stop = cleanup;
      try {
        try { if (session) session.type = 'play-and-record'; } catch (_) {}
        // Raw, hands-free capture: call processing (echo cancellation, noise
        // suppression, automatic gain) tunes the microphone for a phone held
        // to the ear and treats a voice from across the desk as noise.
        // Nothing plays while recording, and the gain setting sets the level.
        stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: false, noiseSuppression: false, autoGainControl: false } });
        if (stopping) { stream.getTracks().forEach(t => t.stop()); return; }
        stream.getAudioTracks()[0].onended = () => interrupted('track-ended');
        context = new AudioContext({ latencyHint: 'interactive' });
        await context.resume();
        processor = context.createScriptProcessor(1024, 1, 1);
        sink = context.createGain();
        sink.gain.value = 0;
        processor.connect(sink);
        sink.connect(context.destination);
        source = context.createMediaStreamSource(stream);
        source.connect(processor);
        context.onstatechange = () => { if (context.state !== 'running') interrupted('context-' + context.state); };
        if (session) session.onstatechange = () => { if (session.state === 'interrupted') interrupted('session-interrupted'); };
        lastBeat = performance.now();
        watchdog = setInterval(() => { if (performance.now() - lastBeat > 3000) interrupted('heartbeat-expired'); }, 500);
        const step = context.sampleRate / targetRate;
        let position = 0, sum = 0, count = 0;
        processor.onaudioprocess = e => {
          const input = e.inputBuffer.getChannelData(0);
          const out = new Int16Array(Math.ceil(input.length / step) + 1);
          let written = 0, energy = 0;
          for (const x of input) {
            energy += x * x; sum += x; count += 1; position += 1;
            if (position >= step) { position -= step; out[written++] = Math.max(-1, Math.min(1, sum / count)) * 32767; sum = 0; count = 0; }
          }
          if (written > 0) post({ type: 'audio', data: encode(out.subarray(0, written)), rms: Math.sqrt(energy / input.length) });
        };
        post({ type: 'started' });
      } catch (error) {
        if (stopping) return;
        cleanup();
        post({ type: 'failed', reason: error.name });
      }
    };
    post({ type: 'ready' });
    </script></body></html>
    """
}

extension WebMicrophone: WKUIDelegate, WKNavigationDelegate {
    public func webView(
        _ webView: WKWebView,
        requestMediaCapturePermissionFor origin: WKSecurityOrigin,
        initiatedByFrame frame: WKFrameInfo,
        type: WKMediaCaptureType,
        decisionHandler: @escaping @MainActor (WKPermissionDecision) -> Void
    ) {
        // The app's microphone permission still applies; this only skips
        // WebKit's per-origin prompt, for the microphone alone.
        decisionHandler(type == .microphone ? .grant : .deny)
    }

    public func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
        isReady = false
        finish(reason: "web content process terminated")
        load()
    }
}

/// Holds the microphone weakly, so the content controller does not retain it.
private final class MessageRelay: NSObject, WKScriptMessageHandler {
    weak var target: WebMicrophone?
    init(_ target: WebMicrophone) { self.target = target }

    func userContentController(_ controller: WKUserContentController, didReceive message: WKScriptMessage) {
        MainActor.assumeIsolated { target?.receive(message.body) }
    }
}
