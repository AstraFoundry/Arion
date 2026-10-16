import pathlib,json,time
import onnxruntime_genai as og
p=pathlib.Path('.workspace/ablation-models/qwen3-0.6b')
print('runtime',og.__version__,flush=True)
if (p/'genai_config.json').exists():
 print((p/'genai_config.json').read_text(),flush=True)
 cfg=og.Config(str(p));cfg.clear_providers()
 t=time.perf_counter();m=og.Model(cfg);print('load',time.perf_counter()-t,flush=True)
 tok=og.Tokenizer(m)
 prompt='<|im_start|>system\nRewrite dictation. Remove filler words and repetitions. Apply spoken self-corrections. Preserve facts. Return only rewritten text.<|im_end|>\n<|im_start|>user\n嗯我们周三开会，不对改成周四下午三点。<|im_end|>\n<|im_start|>assistant\n<think>\n\n</think>\n\n'
 ids=tok.encode(prompt);pr=og.GeneratorParams(m);pr.set_search_options(max_length=len(ids)+96,do_sample=False)
 g=og.Generator(m,pr);t=time.perf_counter();g.append_tokens(ids);print('prefill',time.perf_counter()-t,flush=True)
 while not g.is_done():g.generate_next_token()
 print(tok.decode(g.get_sequence(0)[len(ids):]),time.perf_counter()-t,flush=True)
