---
language:
- en
tags:
- text2text-generation
license: mit
datasets:
- PeacefulData/HyPoradise-v0
library_name: transformers
pipeline_tag: text2text-generation
widget:
  - text: "Generate the correct transcription for the following n-best list of ASR hypotheses: \n\n1. nebode also typically is symphons and an ankle surf leash \n2. neboda is also typically is symphons and an ankle surf leash \n3. nebode also typically is swim fins and an ankle surf leash \n4. neboda also typically is symphons and an ankle surf leash \n5. neboda is also typically is swim fins and an ankle surf leash"
base_model:
- google/flan-t5-base
---

# FLANEC: Exploring FLAN-T5 for Post-ASR Error Correction

## Model Overview

FLANEC is an encoder-decoder model based on FLAN-T5, specifically fine-tuned for post-Automatic Speech Recognition (ASR) error correction, also known as Generative Speech Error Correction (GenSEC). The model utilizes n-best hypotheses from ASR systems to enhance the accuracy and grammaticality of final transcriptions by generating a single corrected output. FLANEC models are trained on diverse subsets of the [HyPoradise dataset](https://huggingface.co/datasets/PeacefulData/HyPoradise-v0), leveraging multiple ASR domains to provide robust, scalable error correction across different types of audio data.

FLANEC was developed for the **GenSEC Task 1 challenge at SLT 2024** - [Challenge website](https://sites.google.com/view/gensec-challenge/home).

## Model Checkpoints

**Cumulative Dataset (CD) Models trained with full fine-tuning:**

- [FLANEC Base CD](https://huggingface.co/morenolq/flanec-base-cd): Base model with ~250 million parameters, fine-tuned for post-ASR correction on cumulative datasets.
- [FLANEC Large CD](https://huggingface.co/morenolq/flanec-large-cd): Large model with ~800 million parameters, fine-tuned for post-ASR correction on cumulative datasets.
- [FLANEC XL CD](https://huggingface.co/morenolq/flanec-xl-cd): Extra-large model with ~3 billion parameters, fine-tuned for post-ASR correction on cumulative datasets.

**Cumulative Dataset (CD) Models trained with Low-Rank Adaptation (LoRA):**

- [FLANEC Base LoRA](https://huggingface.co/morenolq/flanec-base-cd-lora): Base model with ~250 million parameters, fine-tuned with LoRA on cumulative datasets.
- [FLANEC Large LoRA](https://huggingface.co/morenolq/flanec-large-cd-lora): Large model with ~800 million parameters, fine-tuned with LoRA on cumulative datasets.
- [FLANEC XL LoRA](https://huggingface.co/morenolq/flanec-xl-cd-lora): Extra-large model with ~3 billion parameters, fine-tuned with LoRA on cumulative datasets.

## Intended Use

FLANEC is designed for the task of Generative Speech Error Correction (GenSEC). The model is suitable for post-processing ASR outputs to correct grammatical and linguistic errors. The model supports the **English** language.

## Training Details

### Datasets

FLANEC is trained on the [HyPoradise dataset](https://huggingface.co/datasets/PeacefulData/HyPoradise-v0), which contains data from eight ASR domains:

1. **WSJ**: Business and financial news.
2. **ATIS**: Airline travel queries.
3. **CHiME-4**: Noisy speech.
4. **Tedlium-3**: TED talks.
5. **CV-accent**: Accented speech.
6. **SwitchBoard**: Conversational speech.
7. **LRS2**: BBC program audio.
8. **CORAAL**: Accented speech from African American English.

For more details, see the [HyPoradise paper](https://proceedings.neurips.cc/paper_files/paper/2023/hash/6492267465a7ac507be1f9fd1174e78d-Abstract-Datasets_and_Benchmarks.html).

### Training Strategy

The model has been fine-tuned using both full fine-tuning and LoRA (Low-Rank Adaptation) methods. Fine-tuning was performed on multiple model scales, ranging from 250M to 3B parameters. Both single-dataset (SD) and cumulative dataset (CD) training approaches were employed to assess model performance across different ASR domains.

For more information on the training strategy, refer to the SLT 2024 paper.

## Citation

- [arXiv version](https://arxiv.org/abs/2501.12979)
- [IEEEXplore version](https://ieeexplore.ieee.org/document/10832257)

Please use the following citation to reference this work in your research:

```bibtex
@article{quatra_2024_flanec:,
  author = {Moreno La Quatra and Valerio Mario Salerno and Yu Tsao and Sabato Marco Siniscalchi},
  title = {FlanEC: Exploring Flan-T5 for Post-ASR Error Correction},
  journal = {2024 IEEE Spoken Language Technology Workshop (SLT)},
  year = {2024},
  doi = {10.1109/slt61566.2024.10832257},
  url = {https://doi.org/10.1109/slt61566.2024.10832257}
}
```