"""Real accelerated X-ASR streaming and offline control, without reference inputs."""
import importlib.util
import json
import pathlib
import resource
import sys
import time
import wave
import numpy as np
import sherpa_onnx

ROOT = pathlib.Path(__file__).resolve().parents[3]
spec = importlib.util.spec_from_file_location('phonetic_acoustics', ROOT/'tools/evaluation/phonetic/acoustic_worker.py')
shared = importlib.util.module_from_spec(spec)
spec.loader.exec_module(shared)


def decode_online(model, samples):
    stream = model.create_stream()
    segments = []
    for offset in range(0, len(samples), 320):
        stream.accept_waveform(16000, samples[offset:offset+320])
        while model.is_ready(stream): model.decode_stream(stream)
        if model.is_endpoint(stream):
            result = shared.segment_result(model, stream)
            if result['text']: segments.append(result)
            model.reset(stream)
    stream.accept_waveform(16000, np.zeros(6400,dtype=np.float32))
    stream.input_finished()
    while model.is_ready(stream): model.decode_stream(stream)
    result = shared.segment_result(model,stream)
    if result['text']: segments.append(result)
    return dict(text=' '.join(s['text'] for s in segments),segments=segments)


def main():
    online = offline = None
    for line in sys.stdin:
        request=json.loads(line)
        try:
            start=time.perf_counter()
            if request['op']=='load':
                for artifact in request['artifacts']:
                    if shared.digest(artifact['path'])!=artifact['sha256']:raise ValueError('Acoustic artifact digest mismatch')
                online=sherpa_onnx.OnlineRecognizer.from_transducer(**request['online'],num_threads=4,sample_rate=16000,decoding_method='greedy_search',enable_endpoint_detection=True,rule1_min_trailing_silence=2.4,rule2_min_trailing_silence=0.8,rule3_min_utterance_length=20.0,provider='cpu')
                offline=sherpa_onnx.OfflineRecognizer.from_transducer(**request['offline'],num_threads=4,sample_rate=16000,modeling_unit='cjkchar',decoding_method='greedy_search',provider='cpu')
                result=dict(loadMs=(time.perf_counter()-start)*1000,sherpa=sherpa_onnx.__version__,numpy=np.__version__)
            elif request['op']=='infer':
                audio=request['audio']
                if shared.digest(audio['path'])!=audio['sha256']:raise ValueError('Audio digest mismatch')
                with wave.open(audio['path'],'rb') as source:
                    if (source.getframerate(),source.getnchannels(),source.getsampwidth())!=(16000,1,2):raise ValueError('Audio format mismatch')
                    samples=np.frombuffer(source.readframes(source.getnframes()),dtype='<i2').astype(np.float32)/32768
                if abs(len(samples)/16-audio['durationMs'])>1:raise ValueError('Audio duration mismatch')
                start=time.perf_counter();streaming=decode_online(online,samples);streaming['computeMs']=(time.perf_counter()-start)*1000
                start=time.perf_counter();stream=offline.create_stream();stream.accept_waveform(16000,samples);offline.decode_stream(stream)
                result=dict(streaming=streaming,offline=dict(text=stream.result.text.strip(),computeMs=(time.perf_counter()-start)*1000),peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss)
            elif request['op']=='close':
                result=dict(closed=True);print(json.dumps(dict(ok=True,**result)),flush=True);break
            else:raise ValueError('Unknown operation')
            print(json.dumps(dict(ok=True,**result),ensure_ascii=False),flush=True)
        except Exception as error:
            print(json.dumps(dict(ok=False,error=str(error))),flush=True)


if __name__=='__main__':main()
