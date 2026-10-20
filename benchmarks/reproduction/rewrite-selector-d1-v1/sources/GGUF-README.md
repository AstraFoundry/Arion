---
library_name: gguf
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
base_model: LiquidAI/d1-omni-600M
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
- gguf
- llama.cpp
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

# d1-omni-600M-GGUF

d1-omni-600M is a decision model developed by [Liquid AI](https://www.liquid.ai/): it answers named, typed questions over a state (text, JSON, images or audio) in one forward pass, with no generated tokens.

Find more details in the original model card: https://huggingface.co/LiquidAI/d1-omni-600m

## 🏃 How to run d1-omni-600M

Example usage with [llama.cpp](https://github.com/ggml-org/llama.cpp):

```bash
llama-server -hf LiquidAI/d1-omni-600M-GGUF:Q8_0 -b 4096 -ub 4096
```

A question is read in one batch, so `-b` and `-ub` must hold it: 4096 covers any image and states of a few thousand tokens, raise them up to 16384 for longer states.

Then send questions to the `/v1/systemone` endpoint.

### Text

```bash
curl http://127.0.0.1:8080/v1/systemone -H "Content-Type: application/json" -d '{
  "state": "I was charged twice this month, please refund one of them.",
  "questions": {
    "refund":  {"type": "noul", "instructions": "Is the customer asking for a refund?"},
    "team":    {"type": "choice", "instructions": "Which team should handle this?",
                "criteria": {"billing": "Charges, refunds, invoices", "technical": "App or site faults",
                             "fraud": "Suspected unauthorised use"}},
    "urgency": {"type": "score", "instructions": "How urgent is this?",
                "criteria": ["Can wait", "Today", "Blocking the customer now"]}
  }
}'
```

### Image + Text

```bash
curl -sL -o cats.jpg http://images.cocodataset.org/val2017/000000039769.jpg  # two cats on a sofa

curl http://127.0.0.1:8080/v1/systemone -H "Content-Type: application/json" -d @- <<JSON
{
  "state": "Photo attached to a pet-sitting request.",
  "images": ["data:image/jpeg;base64,$(base64 < cats.jpg | tr -d '\n')"],
  "questions": {
    "pet":  {"type": "choice", "instructions": "Which animals are in the photo?",
             "criteria": {"cats": "Cats", "dogs": "Dogs", "birds": "Birds"}},
    "sofa": {"type": "noul", "instructions": "Are the animals on a sofa?"}
  }
}
JSON
```

### Audio + Text

```bash
curl -sL -o jfk.wav https://github.com/ggml-org/whisper.cpp/raw/master/samples/jfk.wav

curl http://127.0.0.1:8080/v1/systemone -H "Content-Type: application/json" -d @- <<JSON
{
  "state": "Recording from a public event.",
  "files": ["data:audio/wav;base64,$(base64 < jfk.wav | tr -d '\n')"],
  "questions": {
    "kind": {"type": "choice", "instructions": "What kind of utterance is this?",
             "criteria": {"request": "A request to do something",
                          "question": "A question asking for information",
                          "speech": "A speech to an audience"}},
    "calm": {"type": "score", "instructions": "How calm is the speaker?",
             "criteria": ["Agitated", "Neutral", "Calm"]}
  }
}
JSON
```

The state can be `null` when the images or the audio are the whole state. A request carries images or one audio clip (up to 30 s), not both.
