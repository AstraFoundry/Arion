"""Audio-only offline transducer control; references stay in the Node evaluator."""
import hashlib
import json
import resource
import sys
import time
import wave

import numpy as np
import sherpa_onnx


def digest(path):
    with open(path, "rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


model = None
for line in sys.stdin:
    request = json.loads(line)
    try:
        if request["op"] == "load":
            for artifact in request["artifacts"]:
                if digest(artifact["path"]) != artifact["sha256"]:
                    raise ValueError("Control artifact checksum mismatch")
            start = time.perf_counter()
            model = sherpa_onnx.OfflineRecognizer.from_transducer(
                **request["files"], num_threads=4, sample_rate=16000,
                modeling_unit=request["modelingUnit"], decoding_method="greedy_search", provider="cpu")
            response = dict(loadMs=(time.perf_counter() - start) * 1000,
                            sherpa=sherpa_onnx.__version__, numpy=np.__version__)
        elif request["op"] == "infer":
            audio = request["audio"]
            if digest(audio["path"]) != audio["sha256"]:
                raise ValueError("Control audio checksum mismatch")
            with wave.open(audio["path"], "rb") as source:
                if (source.getframerate(), source.getnchannels(), source.getsampwidth()) != (16000, 1, 2):
                    raise ValueError("Expected mono PCM16 16kHz")
                samples = np.frombuffer(source.readframes(source.getnframes()), dtype=np.int16).astype(np.float32) / 32768
            if abs(len(samples) / 16 - audio["durationMs"]) > 1:
                raise ValueError("Control duration mismatch")
            start = time.perf_counter()
            stream = model.create_stream()
            stream.accept_waveform(16000, samples)
            model.decode_stream(stream)
            response = dict(text=stream.result.text.strip(), computeMs=(time.perf_counter() - start) * 1000,
                            peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss)
        elif request["op"] == "close":
            model = None
            response = dict(closed=True)
        else:
            raise ValueError("Unknown control operation")
        print(json.dumps(dict(id=request["id"], ok=True, **response), ensure_ascii=False), flush=True)
    except Exception as error:
        print(json.dumps(dict(id=request["id"], ok=False, error=str(error))), flush=True)
