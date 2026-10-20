---
license: other
license_name: youtu-llm
license_link: https://huggingface.co/tencent/Youtu-LLM-2B-GGUF/blob/main/LICENSE.txt
pipeline_tag: text-generation
base_model:
  - tencent/Youtu-LLM-2B
base_model_relation: quantized
---
<div align="center">

# <img src="assets/youtu-llm-logo.png" alt="Youtu-LLM Logo" height="100px">

[📃 License](https://huggingface.co/tencent/Youtu-LLM-2B-GGUF/blob/main/LICENSE.txt) • [💻 Code](https://github.com/TencentCloudADP/youtu-tip/tree/master/youtu-llm) • [📑 Technical Report](https://arxiv.org/abs/2512.24618) • [📊 Benchmarks](#benchmarks) • [🚀 Getting Started](#quickstart)

</div>

## 🎯 Brief Introduction

**Youtu-LLM** is a new, small, yet powerful LLM, contains only 1.96B parameters, supports 128k long context, and has native agentic talents. On general evaluations, Youtu-LLM significantly outperforms SOTA LLMs of similar size in terms of Commonsense, STEM, Coding and Long Context capabilities; in agent-related testing, Youtu-LLM surpasses larger-sized leaders and is truly capable of completing multiple end2end agent tasks. 

**Youtu-LLM** has the following features:
- Type: Autoregressive Causal Language Models with Dense MLA
- Release versions: [Base](https://huggingface.co/tencent/Youtu-LLM-2B-Base) and [Instruct](https://huggingface.co/tencent/Youtu-LLM-2B)
- Number of Parameters: 1.96B
- Number of Layers: 32
- Number of Attention Heads (MLA): 16 for Q/K/V
- MLA Rank: 1,536 for Q, 512 for K/V 
- MLA Dim: 128 for QK Nope, 64 for QK Rope, and 128 for V
- Context Length: 131,072
- Vocabulary Size: 128,256

## 🤗 Model Download
| Model Name  | Description | Download |
| ----------- | ----------- |-----------
| Youtu-LLM-2B-Base  | Base model of Youtu-LLM-2B |🤗 [Model](https://huggingface.co/tencent/Youtu-LLM-2B-Base)|
| Youtu-LLM-2B | Instruct model of Youtu-LLM-2B | 🤗 [Model](https://huggingface.co/tencent/Youtu-LLM-2B)|
| Youtu-LLM-2B-GGUF | Instruct model of Youtu-LLM-2B, in GGUF format | 🤗 [Model](https://huggingface.co/tencent/Youtu-LLM-2B-GGUF)|

<a id="benchmarks"></a>

## 📊 Performance Comparisons
### Instruct Model

# <img src="assets/general_agentic_instruct.png" alt="Comparison between Youtu-LLM-2B and baselines" height="260px">

#### General Benchmarks
| Benchmark | DeepSeek-R1-Distill-Qwen-1.5B | Qwen3-1.7B | SmolLM3-3B | Qwen3-4B | DeepSeek-R1-Distill-Llama-8B | Youtu-LLM-2B |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| **Commonsense Knowledge Reasoning** | | | | | | |
| MMLU-Redux | 53.0% | 74.1% | 75.6% | **83.8%** | <u>78.1%</u> | 75.8% |
| MMLU-Pro | 36.5% | 54.9% | 53.0% | **69.1%** | 57.5% | <u>61.6%</u> |
| **Instruction Following & Text Reasoning** | | | | | | |
| IFEval | 29.4% | 70.4% | 60.4% | **83.6%** | 34.6% | <u>81.2%</u> |
| DROP | 41.3% | 72.5% | 72.0% | <u>82.9%<u> | 73.1% | **86.7%** |
| MUSR | 43.8% | 56.6% | 54.1% | **60.5%** | <u>59.7%</u> | 57.4% |
| **STEM** | | | | | | |
| MATH-500 | 84.8% | 89.8% | 91.8% | **95.0%** | 90.8% | <u>93.7%</u> |
| AIME 24 | 30.2% | 44.2% | 46.7% | **73.3%** | 52.5% | <u>65.4%</u> |
| AIME 25 | 23.1% | 37.1% | 34.2% | **64.2%** | 34.4% | <u>49.8%</u> |
| GPQA-Diamond | 33.6% | 36.9% | 43.8% | **55.2%** | 45.5% | <u>48.0%</u> |
| BBH | 31.0% | 69.1% | 76.3% | **87.8%** | <u>77.8%</u> | 77.5% |
| **Coding** | | | | | | |
| HumanEval | 64.0% | 84.8% | 79.9% | <u>95.4%<u> | 88.1% | **95.9%** |
| HumanEval+ | 59.5% | 76.2% | 74.7% | <u>87.8%</u> | 82.5% | **89.0%** |
| MBPP | 51.5% | 80.5% | 66.7% | **92.3%** | 73.9% | <u>85.0%</u> |
| MBPP+ | 44.2% | 67.7% | 56.7% | **77.6%** | 61.0% | <u>71.7%</u> |
| LiveCodeBench v6 | 19.8% | 30.7% | 30.8% | **48.5%** | 36.8% | <u>43.7%</u> |

#### Agentic Benchmarks
| Benchmark | Qwen3-1.7B | SmolLM3-3B | Qwen3-4B | Youtu-LLM-2B |
| :--- | :---: | :---: | :---: | :---: |
| **Deep Research** | | | | |
| GAIA | 11.4% | 11.7% | <u>25.5%</u> | **33.9%** |
| xbench | 11.7% | 13.9% | <u>18.4%</u> | **19.5%** |
| **Code** | | | | |
| SWE-Bench-Verified | 0.6% | <u>7.2%</u> | 5.7% | **17.7%** |
| EnConda-Bench | 10.8% | 3.5% | <u>16.1%</u> | **21.5%** |
| **Tool** | | | | |
| BFCL V3 | 55.5% | 31.5% | **61.7%** | <u>58.0%</u> |
| τ²-Bench | 2.6% | 9.7% | <u>10.9%</u> | **15.0%** |

<a id="quickstart"></a>

## 🚀 Quick Start
This guide will help you quickly deploy and invoke the **Youtu-LLM-2B** model. This model supports "Reasoning Mode", enabling it to generate higher-quality responses through Chain of Thought (CoT).

### Server Example

**Enable Reasoning Mode (default):**
```bash
./llama-server -m Youtu-LLM-2B-F16.gguf \
  --port 8080 \
  --host 0.0.0.0
```

**Disable Reasoning Mode:**
```bash
./llama-server -m Youtu-LLM-2B-F16.gguf \
  --port 8080 \
  --host 0.0.0.0 \
  --reasoning-budget 0
```

### Key Configuration Details

#### Reasoning Mode Toggle

Controlled via the `--reasoning-budget` parameter:

* **Default (no flag):** Enables Chain of Thought; ideal for complex logic and reasoning tasks. Response includes `reasoning_content` field.
* **`--reasoning-budget 0`:** Disables reasoning; faster response time, suitable for simple conversations.

#### Recommended Decoding Parameters

| Parameter | Reasoning Mode | Normal Mode |
| --- | --- | --- |
| `temperature` | **1.0** (Maintains creativity) | **0.7** (More stable results) |
| `top_p` | 0.95 | 0.8 |
| `top_k` | 20 | 20 |
| `repetition_penalty` | 1.05 | - |

> **Tip:** When using Reasoning Mode, a higher `temperature` helps the model perform deeper, more divergent thinking.

## 📚 Citation

If you find our work useful in your research, please consider citing the following paper:

```bibtex
@article{youtu-llm,
  title={Youtu-LLM: Unlocking the Native Agentic Potential for Lightweight Large Language Models},
  author={Tencent Youtu Lab},
  year={2025},
  eprint={2512.24618},
  archivePrefix={arXiv},
  primaryClass={cs.CL},
  url={https://arxiv.org/abs/2512.24618}, 
}
```