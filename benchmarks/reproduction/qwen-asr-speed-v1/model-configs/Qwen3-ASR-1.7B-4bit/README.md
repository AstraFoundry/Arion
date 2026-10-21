---
license: apache-2.0
tags:
- mlx
- speech-to-text
- speech
- transcription
- asr
- stt
- mlx-audio
library_name: mlx-audio
---
# mlx-community/Qwen3-ASR-1.7B-4bit

This model was converted to MLX format from [`Qwen/Qwen3-ASR-1.7B`](https://huggingface.co/Qwen/Qwen3-ASR-1.7B) using mlx-audio version **0.3.1**.

Refer to the [original model card](https://huggingface.co/Qwen/Qwen3-ASR-1.7B) for more details on the model.

## Use with mlx-audio

```bash
pip install -U mlx-audio
```

### CLI Example:
```bash
python -m mlx_audio.stt.generate --model mlx-community/Qwen3-ASR-1.7B-4bit --audio "audio.wav"
```

### Python Example:
```python        
from mlx_audio.stt.utils import load_model
from mlx_audio.stt.generate import generate_transcription

model = load_model("mlx-community/Qwen3-ASR-1.7B-4bit")
transcription = generate_transcription(
    model=model,
    audio_path="path_to_audio.wav",
    output_path="path_to_output.txt",
    format="txt",
    verbose=True,
)
print(transcription.text)

```
