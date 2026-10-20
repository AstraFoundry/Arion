---
library_name: transformers
pipeline_tag: text-generation
license: other
license_name: lfm1.0
license_link: LICENSE
base_model: LiquidAI/LFM2.5-1.2B-Instruct
tags:
- liquid
- lfm2.5
- qad
- safetensors
---

# LFM2.5-1.2B-Instruct QAD — FP32 source checkpoint

This directory contains the unquantized **FP32 safetensors source weights** for
[LFM2.5-1.2B-Instruct-QAD-Q4_0.gguf](https://huggingface.co/LiquidAI/LFM2.5-1.2B-Instruct-GGUF/blob/main/LFM2.5-1.2B-Instruct-QAD-Q4_0.gguf).
These are the weights after Quantization-Aware Distillation (QAD), before GGUF
conversion and quantization. The architecture and tokenizer are those of
[LiquidAI/LFM2.5-1.2B-Instruct](https://huggingface.co/LiquidAI/LFM2.5-1.2B-Instruct).

The checkpoint is provided for fine-tuning, inspecting weights, and experimenting
with deployment formats. It was optimized for Q4_0 deployment; published QAD Q4_0
benchmark results do not describe direct FP32/BF16 inference or other quantization
formats. See the [QAD release](https://www.liquid.ai/blog/qad).

## Usage

Install current `torch`, `transformers`, and `accelerate`. This package was checked
with Transformers 5.9.0. `dtype="auto"` preserves the stored FP32 weights.

```python
from transformers import AutoModelForCausalLM, AutoTokenizer

repo_id = "LiquidAI/LFM2.5-1.2B-Instruct-GGUF"
tokenizer = AutoTokenizer.from_pretrained(repo_id, subfolder="qad")
model = AutoModelForCausalLM.from_pretrained(
    repo_id, subfolder="qad", dtype="auto", device_map="auto"
)

inputs = tokenizer.apply_chat_template(
    [{"role": "user", "content": "What is 2 + 2?"}],
    tokenize=True, add_generation_prompt=True, return_dict=True, return_tensors="pt",
).to(model.device)
outputs = model.generate(**inputs, max_new_tokens=128)
print(tokenizer.decode(outputs[0, inputs["input_ids"].shape[-1]:], skip_special_tokens=True))
```

## Files and validation

- `model.safetensors`: original FP32 QAD weights.
- `config.json`: the validated source architecture/configuration, including its FP32 dtype.
- `tokenizer.json`, `tokenizer_config.json`: tokenizer files.
- `chat_template.jinja`: exactly the template embedded in the public QAD GGUF.
- `generation_config.json`: defaults from the corresponding public HF model.
- `LICENSE`: the repository's LFM license.
- `provenance.json`: source hash and the validated GGUF reference.

The FP32 weights passed finite-value checks and Transformers loading, generation,
and cache-consistency smoke tests. Converting these weights to F16 GGUF and then
Q4_0 with llama.cpp revision `74ade52741203e5c8f81eaf06a96cb1cfe15f2a3` reproduced
every tensor byte of the released QAD artifact. Applying the same release metadata
also reproduced its complete SHA-256. File names and descriptive GGUF metadata
can otherwise cause different whole-file hashes despite identical tensors.

Preserve FP32 weights when reproducing that conversion: saving a BF16 copy first
changes the source precision. Generation defaults follow the public model;
use `do_sample=False` for deterministic greedy decoding.

## License

This checkpoint is distributed under the same [LFM license](LICENSE) as the
corresponding model release.
