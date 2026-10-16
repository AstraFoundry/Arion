"""Reference-isolated, accelerated acoustic evidence extraction for diagnostics."""
import contextlib
import hashlib
import json
import pathlib
import resource
import sys
import time
import wave

import numpy as np
import pypinyin
import sherpa_onnx
import torch
import allosaurus
from allosaurus.app import read_recognizer


def digest(path):
    with open(path, "rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def segment_result(recognizer, stream):
    result = recognizer.get_result_all(stream)
    data = json.loads(result.as_json_string())
    tokens = list(result.tokens)
    scores = list(result.ys_probs)
    if len(tokens) != len(scores) or any(not np.isfinite(score) for score in scores):
        raise ValueError("Missing or invalid per-token acoustic scores")
    return dict(text=result.text.strip(), tokens=tokens, logScores=scores,
                timestamps=list(result.timestamps), rawResult=data)


def infer(recognizer, phone_model, config, request):
    audio = request["audio"]
    if digest(audio["path"]) != audio["sha256"]:
        raise ValueError("Audio digest mismatch")
    with wave.open(audio["path"], "rb") as source:
        if (source.getframerate(), source.getnchannels(), source.getsampwidth()) != (16000, 1, 2):
            raise ValueError("Expected mono PCM16 at 16kHz")
        samples = np.frombuffer(source.readframes(source.getnframes()), dtype=np.int16).astype(np.float32) / 32768
    if abs(len(samples) / 16 - audio["durationMs"]) > 1:
        raise ValueError("Audio duration mismatch")
    started = time.perf_counter()
    stream = recognizer.create_stream()
    segments = []
    chunk = int(config["chunkMs"] * 16)
    for offset in range(0, len(samples), chunk):
        stream.accept_waveform(16000, samples[offset:offset + chunk])
        while recognizer.is_ready(stream):
            recognizer.decode_stream(stream)
        if recognizer.is_endpoint(stream):
            result = segment_result(recognizer, stream)
            if result["text"]:
                segments.append(result)
            recognizer.reset(stream)
    stream.accept_waveform(16000, np.zeros(int(config["tailPaddingSeconds"] * 16000), dtype=np.float32))
    stream.input_finished()
    while recognizer.is_ready(stream):
        recognizer.decode_stream(stream)
    result = segment_result(recognizer, stream)
    if result["text"]:
        segments.append(result)
    # Keep the engine's segments explicit; this diagnostic does no lexical normalization.
    text = " ".join(segment["text"] for segment in segments)
    asr_ms = (time.perf_counter() - started) * 1000
    started = time.perf_counter()
    pinyin = " ".join(pypinyin.lazy_pinyin(text, style=pypinyin.Style.TONE3, neutral_tone_with_five=True))
    pinyin_ms = (time.perf_counter() - started) * 1000
    started = time.perf_counter()
    phones = phone_model.recognize(audio["path"], lang_id=config["phoneInventory"], topk=1, emit=1.0)
    phone_ms = (time.perf_counter() - started) * 1000
    return dict(asrText=text, segments=segments, pinyin=pinyin, phones=phones,
                asrMs=asr_ms, phoneMs=phone_ms, pinyinMs=pinyin_ms,
                peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss)


def main():
    recognizer = phone_model = config = None
    for line in sys.stdin:
        request = json.loads(line)
        try:
            if request["op"] == "load":
                for artifact in request["artifacts"]:
                    if digest(artifact["path"]) != artifact["sha256"]:
                        raise ValueError("Acoustic artifact digest mismatch")
                config = request["config"]
                torch.set_num_threads(config["threads"])
                torch.set_num_interop_threads(1)
                files = request["asrFiles"]
                started = time.perf_counter()
                recognizer = sherpa_onnx.OnlineRecognizer.from_transducer(
                    **files, num_threads=config["threads"], sample_rate=16000,
                    decoding_method="greedy_search", enable_endpoint_detection=True,
                    rule1_min_trailing_silence=2.4,
                    rule2_min_trailing_silence=config["endpointSilenceSeconds"],
                    rule3_min_utterance_length=20.0, provider="cpu")
                model_path = pathlib.Path(request["phoneModels"]) / config["phoneModel"]
                default_path = pathlib.Path(allosaurus.__file__).parent / "pretrained" / config["phoneModel"]
                if not model_path.is_dir() or default_path.resolve() != model_path.resolve():
                    raise ValueError("Explicit phone model provisioning required")
                with contextlib.redirect_stdout(sys.stderr):
                    phone_model = read_recognizer(config["phoneModel"])
                response = dict(loadMs=(time.perf_counter() - started) * 1000,
                                sherpa=sherpa_onnx.__version__, torch=torch.__version__, numpy=np.__version__,
                                pypinyin=pypinyin.__version__, timing="accelerated_chunk_replay")
            elif request["op"] == "infer":
                if recognizer is None:
                    raise ValueError("Load required")
                response = infer(recognizer, phone_model, config, request)
            elif request["op"] == "close":
                recognizer = phone_model = None
                response = dict(closed=True)
            else:
                raise ValueError("Unsupported acoustic operation")
            print(json.dumps(dict(id=request["id"], ok=True, **response), ensure_ascii=False), flush=True)
        except Exception as error:
            print(json.dumps(dict(id=request["id"], ok=False, error=str(error))), flush=True)


if __name__ == "__main__":
    main()
