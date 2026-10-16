"""Real CPU inference worker for development diagnostics, without native UI claims."""
import gc
import hashlib
import json
import pathlib
import platform
import resource
import sys
import time

import onnxruntime as ort
import onnxruntime_genai as og
from tokenizers import Tokenizer as ReferenceTokenizer

ort.disable_telemetry_events()


def sha256(path):
    h = hashlib.sha256()
    with open(path, 'rb') as source:
        while block := source.read(1024 * 1024):
            h.update(block)
    return h.hexdigest()


def main():
    model = tokenizer = reference_tokenizer = None
    pending = None
    for line in sys.stdin:
        request = json.loads(line)
        try:
            op = request['op']
            if op == 'load':
                for artifact in request['artifacts']:
                    if sha256(artifact['path']) != artifact['sha256']:
                        raise ValueError('artifact checksum mismatch')
                if platform.machine() != 'arm64' and platform.system() == 'Darwin':
                    raise ValueError('macOS requires native arm64 runtime for this experiment')
                t = time.perf_counter()
                config = og.Config(request['modelPath'])
                config.clear_providers()
                model = og.Model(config)
                tokenizer = og.Tokenizer(model)
                reference_tokenizer = ReferenceTokenizer.from_file(str(pathlib.Path(request['modelPath'])/'tokenizer.json'))
                response = {'runtime': og.__version__, 'ortRuntime': ort.__version__, 'providersAvailable': ort.get_available_providers(), 'providerRequested': 'CPUExecutionProvider', 'architecture': platform.machine(), 'loadMs': (time.perf_counter() - t) * 1000}
            elif op in ['generate', 'prepare_prefix', 'finish_prefix']:
                if model is None:
                    raise ValueError('load required')
                ids = tokenizer.encode(request['prompt'])
                if list(ids) != reference_tokenizer.encode(request['prompt'], add_special_tokens=False).ids:
                    raise ValueError('native/reference tokenizer parity failed')
                if len(ids) > request['maxInputTokens']:
                    raise ValueError('input exceeds explicit budget')
                start = time.perf_counter()
                if op == 'finish_prefix':
                    if pending is None or list(ids[:len(pending['ids'])]) != pending['ids']:
                        raise ValueError('prefix token identity mismatch')
                    generator, params = pending['generator'], pending['params']
                    generator.append_tokens(ids[len(pending['ids']):])
                    prefix_tokens = len(pending['ids'])
                    prefix_ms = pending['ms']
                    pending = None
                else:
                    if pending is not None:
                        raise ValueError('outstanding prefix must be finished or closed')
                    params = og.GeneratorParams(model)
                    maximum = request.get('sharedMaxLength', len(ids) + request['maxNewTokens'])
                    params.set_search_options(max_length=maximum, do_sample=False)
                    generator = og.Generator(model, params)
                    generator.append_tokens(ids)
                    prefix_tokens = 0
                    prefix_ms = 0
                prefill = (time.perf_counter() - start) * 1000
                if op == 'prepare_prefix':
                    pending = {'generator': generator, 'params': params, 'ids': list(ids), 'ms': prefill}
                    print(json.dumps({'id':request['id'], 'ok':True, 'prefixTokens':len(ids), 'prefixPrefillMs':prefill}),flush=True)
                    continue
                stream = tokenizer.create_stream()
                events = []
                output_ids = []
                while not generator.is_done() and len(output_ids) < request['maxNewTokens']:
                    generator.generate_next_token()
                    token = int(generator.get_next_tokens()[0])
                    delta = stream.decode(token)
                    output_ids.append(token)
                    events.append({'ms': (time.perf_counter() - start) * 1000, 'tokenId': token, 'delta': delta})
                elapsed = (time.perf_counter() - start) * 1000
                raw = tokenizer.decode(output_ids)
                # EOS is a control token, never ordinary output. No semantic repair occurs here.
                text = raw.removesuffix('<|im_end|>').removesuffix('<|endoftext|>').strip()
                incomplete_thinking = False
                if request.get('enableThinking') and text.startswith('<think>'):
                    if '</think>' in text:
                        text = text.split('</think>', 1)[1].strip()
                    else:
                        text = ''
                        incomplete_thinking = True
                response = {'text': text, 'rawText': raw, 'tokenizerParity': True, 'prefixTokens': prefix_tokens, 'prefixPrefillMs': prefix_ms, 'events': events, 'inputTokens': len(ids), 'outputTokens': len(output_ids), 'prefillMs': prefill, 'generationMs': elapsed, 'firstTokenMs': events[0]['ms'] if events else None, 'truncated': len(output_ids) >= request['maxNewTokens'] or incomplete_thinking, 'incompleteThinking': incomplete_thinking, 'peakRssBytes': resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * (1 if sys.platform == 'darwin' else 1024), 'rssScope': 'process_high_water'}
                del generator, params, stream
            elif op == 'close':
                if pending is not None:
                    pending = None
                    del generator, params
                del tokenizer, model, reference_tokenizer
                tokenizer = model = reference_tokenizer = None
                gc.collect()
                response = {'closed': True}
            else:
                raise ValueError('unsupported operation')
            print(json.dumps({'id': request['id'], 'ok': True, **response}), flush=True)
        except Exception as error:
            print(json.dumps({'id': request.get('id'), 'ok': False, 'error': str(error)}), flush=True)


if __name__ == '__main__':
    main()
