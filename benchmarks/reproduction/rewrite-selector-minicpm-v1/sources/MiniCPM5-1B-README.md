---
license: apache-2.0
language:
  - en
  - zh
library_name: transformers
pipeline_tag: text-generation
tags:
  - minicpm
  - minicpm5
  - llama
  - text-generation
  - long-context
  - tool-calling
  - on-device
  - edge-ai
datasets:
  - openbmb/Ultra-FineWeb
  - openbmb/Ultra-FineWeb-L3
  - openbmb/UltraData-Math
  - openbmb/UltraData-SFT-2605
---

<div align="center">
<img src="https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm_logo.png" width="500em" />
</div>

<p align="center">
<a href="https://arxiv.org/pdf/2506.07900" target="_blank">MiniCPM Tech Report</a> |
<a href="https://modelbest.feishu.cn/wiki/UtWxwcERfiRIpIkBOjuc3h9tn1D" target="_blank">MiniCPM Wiki(Chinese)</a> |
<a href="https://github.com/OpenBMB/MiniCPM" target="_blank">GitHub Repo</a> |
<a href="https://ultradata.openbmb.cn/" target="_blank">UltraData</a> |
<a href="https://github.com/OpenBMB/MiniCPM-Desk-Pet" target="_blank">MiniCPM Desk Pet</a> |
<a href="https://huggingface.co/spaces/openbmb/MiniCPM5-1B-Demo" target="_blank">Online Demo</a>
</p>

<p align="center">
English |
<a href="https://huggingface.co/openbmb/MiniCPM5-1B/blob/main/README-cn.md" target="_blank">中文</a>
</p>

## Highlights

We are releasing **MiniCPM5-1B**, the first model in the **MiniCPM5** series. It is a dense 1B Transformer built for on-device, local deployment, and resource-constrained scenarios, reaching 1B-class open-source SOTA.

🏆 **1B-class open-source SOTA**: compared with strong open-source models in the same size class, MiniCPM5-1B reaches SOTA within this comparison set. Its advantage is most visible in agentic tool use, code generation, and difficult reasoning.

![MiniCPM5-1B capability comparison by domain](https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm5/public_leaderboard_radar_en.png)

🧠 **Hybrid Reasoning**: built-in `<think>` chat template, switch via `enable_thinking`. The same checkpoint serves as both a fast assistant and a deliberate reasoner.

🛠️ **Deployment / Fine-tuning Resources**: the MiniCPM GitHub repo provides single-page cookbooks and Agent Skills for major inference backends and fine-tuning frameworks.

🐱 **Desktop Pet**: a local-LLM desktop pet driven by MiniCPM5-1B.

## Model List

Use this directory to choose the model format that matches your runtime:

- **[MiniCPM5-1B](https://huggingface.co/openbmb/MiniCPM5-1B)** · [ModelScope](https://www.modelscope.cn/models/OpenBMB/MiniCPM5-1B) · BF16 final release (post-trained with RL + OPD) **👈 you are here**
- **[MiniCPM5-1B-SFT](https://huggingface.co/openbmb/MiniCPM5-1B-SFT)** · [ModelScope](https://www.modelscope.cn/models/OpenBMB/MiniCPM5-1B-SFT) · BF16 SFT-only checkpoint (before RL / OPD)
- **[MiniCPM5-1B-Base](https://huggingface.co/openbmb/MiniCPM5-1B-Base)** · [ModelScope](https://www.modelscope.cn/models/OpenBMB/MiniCPM5-1B-Base) · BF16 base checkpoint (pre-training only)
- **[MiniCPM5-1B-GGUF](https://huggingface.co/openbmb/MiniCPM5-1B-GGUF)** · [ModelScope](https://www.modelscope.cn/models/OpenBMB/MiniCPM5-1B-GGUF) · GGUF for llama.cpp / Ollama / LM Studio
- **[MiniCPM5-1B-MLX](https://huggingface.co/openbmb/MiniCPM5-1B-MLX)** · [ModelScope](https://www.modelscope.cn/models/OpenBMB/MiniCPM5-1B-MLX) · MLX / 4bit for Apple Silicon

## Model Information

MiniCPM5-1B has the following features:

- **Type**: Causal Language Model
- **Architecture**: Standard `LlamaForCausalLM`
- **Number of Parameters**: 1,080,632,832
- **Number of Non-Embedding Parameters**: 679,552,512
- **Number of Layers**: 24
- **Number of Attention Heads (GQA)**: 16 for Q and 2 for KV
- **Context Length**: 131,072

## Introduction

MiniCPM5-1B is the first checkpoint in the MiniCPM5 series. It is designed for local assistants, coding agents, tool-use workflows, and reasoning scenarios where a compact model is preferred. The model keeps a small deployment footprint while providing native long-context support and both Think / No Think chat modes through the same checkpoint.

## Evaluation Results

We compare MiniCPM5-1B with strong open-source models in the same size class, including **LFM2.5-1.2B-Thinking**, **Qwen3-0.6B/think** and **Qwen3.5-0.8B/think**. These are capable baselines; within this comparison set, MiniCPM5-1B reaches 1B-class open-source SOTA, with its advantage most visible in tool use, code generation, and difficult reasoning. This makes it a practical choice for local coding agents, tool assistants, and reasoning assistants.

![MiniCPM-5 1B Public Leaderboard](https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm5/public_leaderboard_en.png)

## Training Recipe

The training of MiniCPM5-1B is a full-stack practice of **[UltraData Tiered Data Management](https://arxiv.org/pdf/2602.09003)**, covering three stages: base training, mid-training, and post-training.

During **base training**, the model goes through stable training and decay training to build core language capability and training stability. It then enters **mid-training** to further strengthen target capabilities and adapt to the target data distribution. The training corpus is released alongside the model as [Ultra-FineWeb](https://huggingface.co/datasets/openbmb/Ultra-FineWeb), [Ultra-FineWeb-L3](https://huggingface.co/datasets/openbmb/Ultra-FineWeb-L3), and [UltraData-Math](https://huggingface.co/datasets/openbmb/UltraData-Math).

During **post-training**, we proceed in three steps: **SFT**, **RL**, and **OPD**. We first use **200B tokens of deep-thinking SFT** and **200B tokens of hybrid-thinking SFT** to establish deep-thinking, hybrid-thinking, and general chat abilities; the SFT data is released as [UltraData-SFT-2605](https://huggingface.co/datasets/openbmb/UltraData-SFT-2605). We then train specialized **RL teachers** for math, code, closed-book QA, writing, and related domains, and use **On-Policy Distillation (OPD)** to distill these teachers back into one release model.

![MiniCPM5-1B Training Recipe](https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm5/training_recipe.png)

### What does RL + OPD bring?

**RL + OPD** is a key part of MiniCPM5-1B post-training. On math, code and instruction-following tasks, RL + OPD raises the average score by **↑16 points** while cutting the share of responses that hit the max-tokens budget by **↓29 percentage points**. The figures below show the two-stage Reasoning RL pipeline, score gains, and the drop in overlong responses.

**RL** combines complementary training signals for reasoning, closed-book QA, writing, instruction following, long-context understanding, and general dialogue. Reasoning RL is based on [DAPO-Math-17k](https://huggingface.co/datasets/BytedTsinghua-SIA/DAPO-Math-17k) (inspired by [JustRL](https://arxiv.org/pdf/2512.16649)'s minimalist recipe) and uses a two-stage length schedule to reduce overlong responses while improving reasoning accuracy. We also use [TriviaQA](https://huggingface.co/datasets/mandarjoshi/trivia_qa), [NQ-Open](https://huggingface.co/datasets/google-research-datasets/nq_open), [LongWriter-Zero-RLData](https://huggingface.co/datasets/THU-KEG/LongWriter-Zero-RLData), synthesized verifiable RLVR data, and pair-wise RLHF signals to improve reliability, instruction following, and user experience.

![MiniCPM5-1B RL Two-stage Pipeline](https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm5/rl_two_stage_overview.png)

**OPD** builds on Thinking Machines Lab's [On-Policy Distillation](https://thinkingmachines.ai/blog/on-policy-distillation/) and incorporates implementation improvements from [Rethinking On-Policy Distillation](https://arxiv.org/pdf/2604.13016). In the RL framework, we use reverse KL divergence as the advantage estimate, replacing the original verification-based advantage. At each response position, we take top-k logits from both the student and teacher models, compute reverse KL on the union of the two token sets, and balance the accuracy of the RKL signal with training efficiency. OPD reuses the in-domain prompts used to train each RL teacher as distillation data, so no additional data curation is required.

![MiniCPM5-1B RL + OPD Gains](https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm5/rl_gains.png)

![MiniCPM5-1B RL + OPD Overlong Response Rate Drop](https://raw.githubusercontent.com/OpenBMB/MiniCPM/main/assets/minicpm5/rl_overlong.png)

## Quickstart

### vLLM

```bash
pip install "vllm>=0.21"
vllm serve openbmb/MiniCPM5-1B --port 8000
```

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openbmb/MiniCPM5-1B",
    "messages": [{"role": "user", "content": "Who are you? Please briefly introduce yourself."}],
    "max_tokens": 128,
    "temperature": 0.7
  }'
```

### SGLang

```bash
pip install "sglang[srt]>=0.5.12"
python -m sglang.launch_server --model-path openbmb/MiniCPM5-1B --port 30000
```

```bash
curl http://localhost:30000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openbmb/MiniCPM5-1B",
    "messages": [{"role": "user", "content": "Who are you? Please briefly introduce yourself."}],
    "max_tokens": 128,
    "temperature": 0.7
  }'
```

### Transformers

```bash
pip install -U "transformers>=5.6" accelerate torch
```

```python
from transformers import AutoModelForCausalLM, AutoTokenizer

model_id = "openbmb/MiniCPM5-1B"
tokenizer = AutoTokenizer.from_pretrained(model_id)
model = AutoModelForCausalLM.from_pretrained(
    model_id,
    torch_dtype="auto",
    device_map="auto",
)

messages = [{"role": "user", "content": "Who are you? Please briefly introduce yourself."}]
inputs = tokenizer.apply_chat_template(
    messages,
    tokenize=True,
    add_generation_prompt=True,
    enable_thinking=False,
    return_dict=True,
    return_tensors="pt",
).to(model.device)

outputs = model.generate(**inputs, max_new_tokens=128)
print(tokenizer.decode(outputs[0][inputs["input_ids"].shape[-1]:], skip_special_tokens=True))
```

Recommended chat template sampling:

| Mode | Recommended params | Enable |
| --- | --- | --- |
| **Think** | `temperature=0.9, top_p=0.95` | `enable_thinking=True` |
| **No Think** | `temperature=0.7, top_p=0.95` | `enable_thinking=False` |

## Tool Calling

For tool / function calling, **SGLang is the recommended backend**. MiniCPM5-1B emits XML-style tool calls and SGLang's built-in `minicpm5` parser converts them to OpenAI-compatible `tool_calls` natively:

```bash
python -m sglang.launch_server --model-path openbmb/MiniCPM5-1B --port 30000 \
    --tool-call-parser minicpm5      # or: --tool-call-parser auto
```

## GitHub Cookbooks and Agent Skills

MiniCPM5-1B uses the **standard `LlamaForCausalLM` architecture**, so mainstream inference engines can load it directly: **no custom kernels, no model-code fork**. For step-by-step deployment and fine-tuning instructions, use the GitHub cookbooks below. Agent Skills are linked as GitHub resources for users working with Cursor / Claude Code style coding agents.

### Deployment

| Backend | Model format / use case | Cookbook | Agent Skill |
| --- | --- | --- | --- |
| Transformers | BF16 / FP16 local Python inference, GPU + CPU | [transformers.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/transformers.md) | [minicpm5-deploy-transformers](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-transformers/SKILL.md) |
| vLLM | BF16 / FP16 OpenAI server | [vllm.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/vllm.md) | [minicpm5-deploy-vllm](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-vllm/SKILL.md) |
| SGLang | BF16 / FP16 OpenAI server, recommended for tool calling | [sglang.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/sglang.md) | [minicpm5-deploy-sglang](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-sglang/SKILL.md) |
| llama.cpp | GGUF local inference, CPU/GPU | [llama_cpp.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/llama_cpp.md) | [minicpm5-deploy-llama-cpp](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-llama-cpp/SKILL.md) |
| Ollama | GGUF local on-device runtime | [ollama.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/ollama.md) | [minicpm5-deploy-ollama](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-ollama/SKILL.md) |
| LM Studio | GGUF Mac desktop app and OpenAI server | [lmstudio.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/lmstudio.md) | [minicpm5-deploy-lmstudio](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-lmstudio/SKILL.md) |
| MLX | MLX / 4bit local inference on Apple Silicon | [mlx.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/mlx.md) | [minicpm5-deploy-mlx](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-mlx/SKILL.md) |
| ArcLight | GGUF local on-device, CPU, Desktop & Server | [arclight.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/deployment/arclight.md) | [minicpm5-deploy-arclight](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-deploy-arclight/SKILL.md) |

### Fine-tuning

| Framework | Use case | Cookbook | Agent Skill |
| --- | --- | --- | --- |
| TRL + PEFT | LoRA / SFT fine-tuning | [trl.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/finetune/trl.md) | [minicpm5-finetune-trl](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-finetune-trl/SKILL.md) |
| LLaMA-Factory | Fine-tuning | [llamafactory.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/finetune/llamafactory.md) | [minicpm5-finetune-llamafactory](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-finetune-llamafactory/SKILL.md) |
| ms-swift | Fine-tuning | [ms_swift.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/finetune/ms_swift.md) | [minicpm5-finetune-ms-swift](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-finetune-ms-swift/SKILL.md) |
| unsloth | Fine-tuning | [unsloth.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/finetune/unsloth.md) | [minicpm5-finetune-unsloth](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-finetune-unsloth/SKILL.md) |
| xtuner | Fine-tuning | [xtuner.md](https://github.com/OpenBMB/MiniCPM/blob/main/docs/finetune/xtuner.md) | [minicpm5-finetune-xtuner](https://github.com/OpenBMB/MiniCPM/blob/main/skills/minicpm5-finetune-xtuner/SKILL.md) |

### Other Supported Frameworks

In addition to the deployment and fine-tuning frameworks listed above, MiniCPM5-1B is also supported by FlagOS for multi-chip deployment.

#### FlagOS Overview

To enable large-scale deployment across different AI chips, Beijing Zhiyuan Research Institute, together with numerous research institutions, chip manufacturers, system vendors, and algorithm and software organizations both domestically and internationally, jointly initiated and established the FlagOS Open Source Community.

The FlagOS community is dedicated to building a unified, open-source system software stack for various AI chips, encompassing core open-source projects such as a large-scale operator library, a unified AI compiler, parallel training and inference frameworks, and a unified communication library. It aims to create an open technology ecosystem connecting the “model-system-chip” layers. By enabling “develop once, deploy across chips”, FlagOS unlocks the computational potential of hardware, breaks down the ecosystem silos between different chip software stacks, and effectively reduces migration costs for developers.The FlagOS community fosters an AI hardware and software ecosystem, overcomes single-vendor closed-source monopolies, promotes widespread deployment of AI hardware technologies, and is committed to rooted in China while embracing global collaboration.

Official website express: [https://flagos.io](https://flagos.io/)

<details>
<summary>FlagOS multi-chip support and usage</summary>

#### FlagOS: Supporting Multiple AI Chips

Thanks to FlagOS’s unified multi-chip AI system software stack, MiniCPM5-1B was adapted to 4–5 different AI chips in an extremely short time. Currently, the multi-chip version of MiniCPM5-1B has been released on FlagRelease, FlagOS’s platform for automatic migration, adaptation, and deployment of large models across multi-architecture AI chips. Details are as follows:

|Vendor|ModelScope|Huggingface|
|---|---|---|
|Nvidia|[MiniCPM5-1B-nvidia-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-nvidia-FlagOS)|[MiniCPM5-1B-nvidia-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-nvidia-FlagOS)|
|Hygon|[MiniCPM5-1B-hygon-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-hygon-FlagOS)|[MiniCPM5-1B-hygon-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-hygon-FlagOS)|
|Metax|[MiniCPM5-1B-metax-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-metax-FlagOS)|[MiniCPM5-1B-metax-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-metax-FlagOS)|
|Iluvatar|[MiniCPM5-1B-iluvatar-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-iluvatar-FlagOS)|[MiniCPM5-1B-iluvatar-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-iluvatar-FlagOS)|
|Zhenwu|[MiniCPM5-1B-zhenwu-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-zhenwu-FlagOS)|[MiniCPM5-1B-zhenwu-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-zhenwu-FlagOS)|
|Mthreads|[MiniCPM5-1B-mthreads-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-mthreads-FlagOS)|[MiniCPM5-1B-mthreads-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-mthreads-FlagOS)|
|Kunlunxin|[MiniCPM5-1B-kunlunxin-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-kunlunxin-FlagOS)|[MiniCPM5-1B-kunlunxin-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-kunlunxin-FlagOS)|
|Ascend|[MiniCPM5-1B-ascend-FlagOS](https://modelscope.cn/models/FlagRelease/MiniCPM5-1B-ascend-FlagOS)|[MiniCPM5-1B-ascend-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-ascend-FlagOS)|
|ARM-v9|[MiniCPM5-1B-Armv9-FlagOS](https://modelscope.cn/models/FlagRelease/MiniCPM5-1B-Armv9-FlagOS)|[MiniCPM5-1B-Armv9-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-Armv9-FlagOS)|

#### FlagOS Usage

##### FlagOS Performance Acceleration on Nvidia

###### From FlagRelease (**Recommendation**)

FlagRelease is a platform developed by the FlagOS team for automatic migration, adaptation, and deployment of large models across multi-architecture AI chips. The multi-chip version of MiniCPM5-1B has already been released on FlagRelease. All necessary software packages are pre-installed on the platform, so users do not need to install anything.

###### FlagRelease Image Key Versions

###### FlagRelease Quick Start

|Vendor|ModelScope|Huggingface|
|---|---|---|
|Nvidia|[MiniCPM5-1B-nvidia-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-nvidia-FlagOS)|[MiniCPM5-1B-nvidia-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-nvidia-FlagOS)|
|Hygon|[MiniCPM5-1B-hygon-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-hygon-FlagOS)|[MiniCPM5-1B-hygon-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-hygon-FlagOS)|
|Metax|[MiniCPM5-1B-metax-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-metax-FlagOS)|[MiniCPM5-1B-metax-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-metax-FlagOS)|
|Iluvatar|[MiniCPM5-1B-iluvatar-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-iluvatar-FlagOS)|[MiniCPM5-1B-iluvatar-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-iluvatar-FlagOS)|
|Zhenwu|[MiniCPM5-1B-zhenwu-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-zhenwu-FlagOS)|[MiniCPM5-1B-zhenwu-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-zhenwu-FlagOS)|
|Mthreads|[MiniCPM5-1B-mthreads-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-mthreads-FlagOS)|[MiniCPM5-1B-mthreads-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-mthreads-FlagOS)|
|Kunlunxin|[MiniCPM5-1B-kunlunxin-FlagOS](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-kunlunxin-FlagOS)|[MiniCPM5-1B-kunlunxin-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-kunlunxin-FlagOS)|
|Ascend|[MiniCPM5-1B-ascend-FlagOS](https://modelscope.cn/models/FlagRelease/MiniCPM5-1B-ascend-FlagOS)|[MiniCPM5-1B-ascend-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-ascend-FlagOS)|
|ARM-v9|[MiniCPM5-1B-Armv9-FlagOS](https://modelscope.cn/models/FlagRelease/MiniCPM5-1B-Armv9-FlagOS)|[MiniCPM5-1B-Armv9-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-Armv9-FlagOS)|

###### From Scratch

- Dependencies: Python 3.12, GLIBC 2.39, GLIBCXX 3.4.33, CXXABI 1.3.15

###### Vllm Version

###### Installing the FlagOS Operator Library

Official Repository: https://github.com/flagos-ai/FlagGems

```PowerShell
pip install flag-gems==4.2.1rc0
pip install triton==3.5.1
```

###### Activating Acceleration

You can enable flagGems acceleration by adding the import of flagGems in the source code of vllm where inference is performed.

```Bash
import flag_gems
flag_gems.enable(record=True, once=True, path="/root/gems.txt")
```

```PowerShell
vllm serve ${model_path} \
--trust-remote-code \
--dtype bfloat16 \
--enforce-eager \
--port ${Port} \
--served-model-name ${model_name} \
--gpu-memory-utilization 0.85
```

##### Using FlagOS Unified Multi-Chip Backend Plugin

[**vllm-plugin-FL**](https://github.com/flagos-ai/vllm-plugin-FL) is a plugin built for the vLLM inference/service framework. Developed on top of FlagOS’s unified multi-chip backend, it is designed to extend vLLM’s capabilities and performance across a variety of hardware environments.

###### Using vllm-plugin-FL

|Vendor|From Scratch|From FlagRelease||
|---|---|---|---|
|Nvidia|[vllm-plugin-FL/MiniCPM5-1B](https://github.com/flagos-ai/vllm-plugin-FL/blob/main/examples/minicpm/README.md)|[MiniCPM5-1B-ModelScope](https://www.modelscope.cn/models/FlagRelease/MiniCPM5-1B-nvidia-FlagOS)|[MiniCPM5-1B-nvidia-FlagOS](https://huggingface.co/FlagRelease/MiniCPM5-1B-nvidia-FlagOS)|

</details>

## Desktop Pet

We also ship **[OpenBMB/MiniCPM-Desk-Pet](https://github.com/OpenBMB/MiniCPM-Desk-Pet)**, a desktop pet driven locally by MiniCPM5-1B. It supports Apple Silicon / NVIDIA GPU / CPU paths, can work with coding agents such as Cursor, Claude Code, and Codex, and supports LoRA persona switching.

<a href="https://youtu.be/Ee0slMW8SEk"><img src="https://img.youtube.com/vi/Ee0slMW8SEk/0.jpg" alt="MiniCPM Desk Pet video demo" width="720"></a>

## Limitations and Responsible Use

MiniCPM5-1B is a language model that generates content based on learned statistical patterns from training data. It may produce inaccurate, biased, or unsafe outputs, and generated content should be reviewed and verified before use in high-stakes settings.

Users are responsible for evaluating outputs, applying appropriate safeguards, and complying with applicable laws, regulations, and platform policies.

## License

This repository and MiniCPM model weights are released under the [Apache-2.0](https://github.com/OpenBMB/MiniCPM/blob/main/LICENSE) License.

## Citation

Please cite our paper if you find our work valuable:

```bibtex
@article{minicpm4,
  title={Minicpm4: Ultra-efficient llms on end devices},
  author={MiniCPM, Team},
  journal={arXiv preprint arXiv:2506.07900},
  year={2025}
}
```
