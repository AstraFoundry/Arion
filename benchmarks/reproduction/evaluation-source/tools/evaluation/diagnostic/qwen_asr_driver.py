"""Offline Qwen3-ASR adapter: real audio decoding after a realtime fixture wait."""
import hashlib
import json
import pathlib
import platform
import resource
import subprocess
import sys
import time
import wave

import numpy as np
import sherpa_onnx


def hash_file(path):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        while block := f.read(1024 * 1024):
            h.update(block)
    return h.hexdigest()


def main():
    directory = pathlib.Path(sys.argv[1])
    recognizer = None
    artifacts = None
    for line in sys.stdin:
        request = json.loads(line)
        result = {'protocol': 1, 'requestId': request['requestId'], 'execution': 'live'}
        op = request['op']
        if op == 'hello':
            artifacts = request['artifacts']
            result.update(artifacts=artifacts, hardware=dict(os=sys.platform, cpu=subprocess.check_output(['sysctl', '-n', 'machdep.cpu.brand_string'], text=True).strip(), powerMode=subprocess.check_output(['pmset', '-g'], text=True).strip(), ramBytes=int(subprocess.check_output(['sysctl', '-n', 'hw.memsize'], text=True))), runtime=dict(version=sherpa_onnx.__version__, buildFlags='official macOS arm64 CPU Python wheel; Qwen3-ASR int8; threads=4; greedy; no hotwords', timer='time.perf_counter monotonic', scheduler='batch_decode_after_realtime_fixture_wait; no streaming emissions or native preview'))
        elif op == 'infer':
            if artifacts is None or request['variant']['features'] != dict(memory=False, ranker='source', correction='none') or request['pipeline']['asr'] != 'qwen3-asr-06' or request['pipeline']['threads'] != 4 or request['cacheState'] != 'warm':
                raise ValueError('Unsupported treatment')
            audio = request['input']['audio']
            if hash_file(audio['path']) != audio['sha256']:
                raise ValueError('Audio digest mismatch')
            t = time.perf_counter()
            load = 0
            if recognizer is None:
                recognizer = sherpa_onnx.OfflineRecognizer.from_qwen3_asr(conv_frontend=str(directory/'conv_frontend.onnx'), encoder=str(directory/'encoder.int8.onnx'), decoder=str(directory/'decoder.int8.onnx'), tokenizer=str(directory/'tokenizer'), num_threads=4, provider='cpu', max_total_len=512, max_new_tokens=128, temperature=1e-6, seed=11)
                load = (time.perf_counter() - t) * 1000
            with wave.open(audio['path'], 'rb') as f:
                if f.getframerate() != 16000 or f.getnchannels() != 1 or f.getsampwidth() != 2:
                    raise ValueError('Requires PCM16 mono 16kHz')
                samples = np.frombuffer(f.readframes(f.getnframes()), dtype=np.int16).astype(np.float32) / 32768
            duration = len(samples) / 16
            if abs(duration - audio['durationMs']) > 1:
                raise ValueError('Audio duration mismatch')
            start = time.perf_counter()
            time.sleep(duration / 1000)
            decode_start = time.perf_counter()
            stream = recognizer.create_stream()
            stream.accept_waveform(16000, samples)
            recognizer.decode_stream(stream)
            text = stream.result.text
            final = (time.perf_counter() - start) * 1000
            decode = (time.perf_counter() - decode_start) * 1000
            result.update(effectiveFeatures=request['variant']['features'],effectivePipeline=request['pipeline'], cacheState='warm', rawText=text, selectedText=text, finalText=text, candidates=[dict(id='qwen-asr-final', text=text, source='recognizer')], rankedIds=['qwen-asr-final'], events=[dict(ms=final,text=text,final=True)], timings=dict(speechStartMs=0,speechEndMs=duration,firstPartialMs=None,finalMs=final,reviewMs=final,rankerMs=None,loadMs=load,computeMs=decode+load), resources=dict(peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss))
        elif op == 'cleanup':
            recognizer = None
            result.update(cleanupComplete=True)
        else:
            raise ValueError('Unsupported operation')
        print(json.dumps(result), flush=True)


if __name__ == '__main__':
    main()
