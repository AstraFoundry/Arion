---
library_name: transformers
license: other
license_name: lfm1.0
license_link: LICENSE
language:
- en
- de
- es
- fr
- it
- nl
- pl
- pt
- ar
- hi
- ja
- ru
- tr
- vi
- zh
pipeline_tag: image-text-to-text
base_model: LiquidAI/LFM2.5-Encoder-350M
tags:
- liquid
- lfm2.5
- edge
- decision
- classification
- calibration
- system-one
- multimodal
- vision
- audio
- decision-model
---

<div align="center">
  <img
    src="https://cdn-uploads.huggingface.co/production/uploads/61b8e2ba285851687028d395/2b08LKpev0DNEk6DlnWkY.png"
    alt="Liquid AI"
    style="width: 100%; max-width: 100%; height: auto; display: inline-block; margin-bottom: 0.5em; margin-top: 0.5em;"
  />
  <div style="display: flex; justify-content: center; gap: 0.5em; margin-bottom: 1em;">
    <a href="https://playground.liquid.ai/"><strong>Try LFM</strong></a> •
    <a href="https://docs.liquid.ai/lfm/getting-started/welcome"><strong>Docs</strong></a> •
    <a href="https://discord.com/invite/liquid-ai"><strong>Discord</strong></a>
  </div>
</div>

# d1-omni-600M

d1-omni-600M is a 600M parameter **decision model** built on [LFM2.5-Encoder-350M](https://huggingface.co/LiquidAI/LFM2.5-Encoder-350M).
You give it a state (text or JSON, with images or a voice clip) and a set of named questions. It returns typed
answers with **zero output tokens**: every answer is read directly from the model's distribution over the
options, with no generation and no parsing.

- **Vision-language**: text and images (tiled for large frames, several images per state) in a single forward pass.
- **Audio-language**: text and up to 30 s of speech in a single forward pass.
- **Edge-sized**: 587M parameters: a 381M shared trunk and decision head, a 94M vision encoder and a
  112M audio encoder. Every modality runs the same trunk weights.

Find more information about open d1 in our [blog post](https://www.liquid.ai/blog/open-d1).

![image](https://cdn-uploads.huggingface.co/production/uploads/61b8e2ba285851687028d395/r2H7UlZ_m48SYAhZWIHiu.png)

## 🗒️ Model Details

| Model | Parameters | Description |
|---|---|---|
| [LFM2.5-Encoder-350M](https://huggingface.co/LiquidAI/LFM2.5-Encoder-350M) | 350M | General-purpose encoder model (base) |
| **[d1-omni-600M](https://huggingface.co/LiquidAI/d1-omni-600M)** | 587M | Post-trained for single-pass decisions over text, images and speech |

- **Total parameters**: 587M
- **Vision encoder**: SigLIP2 vision tower from [LFM2.5-VL-450M](https://huggingface.co/LiquidAI/LFM2.5-VL-450M)
- **Audio encoder**: 17-layer FastConformer
- **Context length**: 16,384 tokens (text, image and audio positions together); with images, the state and question
  text is cut to 896 tokens, as trained
- **Vocabulary size**: 65,536

We recommend d1-omni-600M wherever a pipeline needs a yes/no, a pick from named options, or a rating:
routing and triage, moderation, intent and topic classification, voice-command routing, extraction checks,
reranking, agent guardrails, and visual inspection. It is not a chat model and does not write text.

> [!WARNING]
> ⚠️ Audio capabilities were trained on requests between an English speaker and an assistant. Training tasks include what kind of utterance
> it is, its topic, and what the speaker wants. Clips are also cut at 30 seconds.


## 🏃 How to use

Install the dependencies (requires `transformers>=5.15`):

```bash
pip install "transformers>=5.15" torch torchvision pillow soundfile
```

The model ships its own code, so load it with `trust_remote_code=True`:

```python
import io
from urllib.request import urlopen

import soundfile as sf
import torch
from transformers import AutoModel
from transformers.image_utils import load_image

device = "cuda" if torch.cuda.is_available() else "mps" if torch.backends.mps.is_available() else "cpu"
dtype = torch.float32 if device == "cpu" else torch.float16
model = AutoModel.from_pretrained("LiquidAI/d1-omni-600M", trust_remote_code=True, dtype=dtype).to(device)

# Text: several named questions over one state, answered in one pass
questions = {
    "refund": {
        "type": "noul",
        "instructions": "Is the customer asking for a refund?",
    },
    "team": {
        "type": "choice",
        "instructions": "Which team should handle this?",
        "criteria": {
            "billing": "Charges, refunds, invoices",
            "technical": "App or site faults",
            "fraud": "Suspected unauthorised use",
        },
    },
    "urgency": {
        "type": "score",
        "instructions": "How urgent is this?",
        "criteria": ["Can wait", "Today", "Blocking the customer now"],
    },
}
print(model.system_one("I was charged twice this month, please refund one of them.", questions))

# Image: the photo is the whole state
image = load_image("http://images.cocodataset.org/val2017/000000039769.jpg")  # two cats on a sofa
cats = {
    "type": "choice",
    "instructions": "How many cats are there?",
    "criteria": {"one": "One", "two": "Two", "more": "Three or more"},
}
print(model.system_one(None, {"cats": cats}, images=[image]))

# Text + audio: one 16 kHz mono clip
url = "https://huggingface.co/datasets/Narsil/asr_dummy/resolve/main/1.flac"
audio, rate = sf.read(io.BytesIO(urlopen(url).read()), dtype="int16")
topic = {
    "type": "choice",
    "instructions": "What is the speaker talking about?",
    "criteria": {"food": "Food and meals", "travel": "Travel and transport", "weather": "The weather"},
}
print(model.system_one("Voice note from a user.", {"topic": topic}, audio=audio))

# Batch: many requests in one call
tickets = ["Where is my parcel? It was due Monday.", "The app crashes when I open settings."]
print(model.system_one_batch([(t, {"team": questions["team"]}) for t in tickets]))
```

| call | |
|---|---|
| `system_one(state, questions, images=None, audio=None)` | Named questions over one state, and its images or its audio clip. The media are encoded once for all questions. |
| `system_one_batch([(state, questions[, images[, audio]]), ...])` | Many requests, batched together. |
| `probabilities(state, questions, images=None, audio=None)` | The raw distributions, in option order. |

A state is a string, any JSON value, or `None` when the images or the audio are the whole state. `images` is a PIL
image or a list of them; `audio` is one 16 kHz mono clip as int16 or float samples. A request
carries images or audio, not both: passing both raises a `ValueError`.

The model was trained in float32. On GPUs, float16 is faster and keeps the float32 answers: on our checks it
gave the same top answer on every text (243), image (214) and audio (416) row. Avoid bfloat16, which changed the
top answer on 0.8% of text and 1.7% of audio rows.

### Questions and answers

Questions follow the Decision Index schema: `type`, `instructions`, and `criteria`.

| `type` | `criteria` | answer fields |
|---|---|---|
| `noul`: yes or no | optional: `{"true": "...", "false": "..."}` to define each side | `noul`: P(yes) |
| `choice`: one of named options | `{name: description}` | `choice`, `confidence`, `probabilities` |
| `score`: 2 to 10 ordered levels | a list of level descriptions, lowest first | `score` (the expected level), `confidence`, `probabilities`, `legend` |

With audio, questions are written the way the audio questions were trained: choice options by their description
(`option_000: ...`), yes/no as plain yes or no. Answers still come back under your option names.

Each call returns `{"answers": {name: answer}, "usage": {"input_tokens": n, "output_tokens": 0}}`, where
`input_tokens` counts every position the trunk read.

Text answers are calibrated with per-type temperatures stored in `config.json`; image and audio answers are the
model's softmax as trained.

## ⚡ Speed

We don't report inference numbers for d1-omni-600M as it is an early research release and is under active development.

## 📊 Performance

### Decision Index 0.2.1

We scored d1-omni-600M and d1-3B with the official scorer (not leaderboard submissions). All other rows come from the
public leaderboard v0.2.1.

| Model | Size | Decision Index | Knowledge | Language | Retrieval | Tools | Arts |
|---|---:|---:|---:|---:|---:|---:|---:|
| Winnow-12B | 12B | 50.02 | 33.8 | 56.0 | 54.0 | 71.0 | 30.0 |
| d1-3B | 3B | 48.57 | 23.8 | 56.4 | 52.8 | **74.5** | **36.3** |
| Decider 35B-A3B | 36B | 47.11 | 31.8 | 55.5 | 54.7 | 56.5 | 32.6 |
| JPT-9B | 9.7B | 46.89 | 31.7 | 56.7 | 44.6 | 67.0 | 28.6 |
| Decision 1.0 Lux | 9.7B | 43.49 | 30.9 | 48.0 | 50.0 | 57.2 | 26.4 |
| JPT-4B | 4.7B | 43.04 | 28.7 | 52.5 | 45.0 | 57.2 | 25.8 |
| Jet v6.2 | 4.7B | 42.60 | 28.7 | 43.9 | 48.2 | 62.9 | 27.0 |
| Decider 4B | 4.7B | 40.70 | 25.7 | 46.0 | 44.7 | 58.6 | 25.0 |
| Winnow-E4B | 8.0B | 39.89 | 22.3 | 45.1 | 43.8 | 62.5 | 22.8 |
| Decider 2B | 2.3B | 28.97 | 14.9 | 32.6 | 37.3 | 42.4 | 14.6 |
| **d1-omni-600M** | **587M** | **15.95** | 8.3 | 12.9 | 35.0 | 15.1 | 6.8 |

### Benchmarks as decisions

Besides the Decision Index, we added a few other internal evaluations based on public benchmarks. HelpSteer2 is
left out: it may overlap with d1-omni-600M's training data.

| Benchmark | d1-omni-600M | d1-3B | Decider 4B | Decider 2B |
|---|---:|---:|---:|---:|
| SQuAD 2.0 | 74.0 | **85.3** | 76.0 | 67.7 |
| Civil Comments | **95.8** | 93.0 | 92.8 | 93.6 |
| MASSIVE intent | 86.1 | 87.3 | **88.3** | 81.1 |
| PubMedQA | 61.3 | **66.0** | 63.3 | 65.7 |
| BoolQ | 77.7 | 86.7 | **89.0** | 87.3 |
| XNLI | 74.7 | 85.0 | **88.6** | 85.0 |
| PAWS-X | **79.5** | 76.9 | 69.8 | 59.5 |
| **Mean** | 78.4 | **82.9** | 81.1 | 77.1 |

d1-omni-600M also scores 76.9 on [Fast Decisions](https://huggingface.co/datasets/fastino/fast-decisions) (dev split).

## 📬 Contact

- Got questions or want to connect? [Join our Discord community](https://discord.com/invite/liquid-ai)
- If you are interested in custom solutions with edge deployment, please contact [our sales team](https://www.liquid.ai/contact).

## Citation

```bibtex
@article{liquidAI2026opend1,
  author  = {Liquid AI},
  title   = {Open d1: Edge decision models for text, vision, and audio},
  journal = {Liquid AI Blog},
  year    = {2026},
  note    = {https://www.liquid.ai/blog/d1-open},
}
```

```bibtex
@article{liquidai2025lfm2,
  title   = {LFM2 Technical Report},
  author  = {Liquid AI},
  journal = {arXiv preprint arXiv:2511.23404},
  year    = {2025}
}
```