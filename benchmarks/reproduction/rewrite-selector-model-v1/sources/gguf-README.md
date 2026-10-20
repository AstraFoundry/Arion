---
license: apache-2.0
library_name: transformers
tags:
- language
- granite-4.0
- gguf
base_model:
- ibm-granite/granite-4.0-1b
---

### Granite 4.0 1b (GGUF)

> [!NOTE]
> This repository contains models that have been converted to the GGUF format with various quantizations from an IBM Granite base model.
>
> Please reference the base model's full model card here:
> https://huggingface.co/ibm-granite/granite-4.0-1b

### Known Issues

This model often uses the full numerical range of a 32-bit float (`f32`), so variants with smaller numerical ranges may run into precision errors at inference. The `F16` variant is known to fail on many hardware combinations.

**The recommended full-precision variant is `bf16`**.