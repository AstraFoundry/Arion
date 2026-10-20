---
license: apache-2.0
library_name: transformers
tags:
- language
- granite-4.0
base_model:
- ibm-granite/granite-4.0-1b-base
---

# Granite-4.0-1B

**Model Summary:**
Granite-4.0-1B is a lightweight instruct model finetuned from *Granite-4.0-1B-Base* using a combination of open source instruction datasets with permissive license and internally collected synthetic datasets. This model is developed using a diverse set of techniques including supervised finetuning, reinforcement learning, and model merging. 

- **Developers:** Granite Team, IBM
- **HF Collection:** [Granite 4.0 Nano Language Models HF Collection](https://huggingface.co/collections/ibm-granite/granite-40-nano-language-models-68e5775c80b60e43b72cfa16)
- **GitHub Repository:** [ibm-granite/granite-4.0-nano-language-models](https://github.com/ibm-granite/granite-4.0-nano-language-models)
- **Website**: [Granite Docs](https://www.ibm.com/granite/docs/) 
- **Release Date**: October 28, 2025
- **License:** [Apache 2.0](https://www.apache.org/licenses/LICENSE-2.0)

**Supported Languages:** 
English, German, Spanish, French, Japanese, Portuguese, Arabic, Czech, Italian, Korean, Dutch, and Chinese. Users may fine-tune Granite 4.0 Nano models to support languages beyond those included in this list.

**Intended use:** 
Granite 4.0 Nano instruct models feature strong instruction following capabilities bringing advanced AI capabilities within reach for on-device deployments and research use cases. Additionally, their compact size makes them well-suited for fine-tuning on specialized domains without requiring massive compute resources.

*Capabilities*
* Summarization
* Text classification
* Text extraction
* Question-answering
* Retrieval Augmented Generation (RAG)
* Code related tasks
* Function-calling tasks
* Multilingual dialog use cases
* Fill-In-the-Middle (FIM) code completions

<!-- <todo>Need to test the examples. (especially the tool calling and RAG ones)</todo>
 -->
 
**Generation:** 
This is a simple example of how to use Granite-4.0-1B model.

Install the following libraries:

```shell
pip install torch torchvision torchaudio
pip install accelerate
pip install transformers
```
Then, copy the snippet from the section that is relevant for your use case.

```python
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer

device = "cuda"
model_path = "ibm-granite/granite-4.0-1b"
tokenizer = AutoTokenizer.from_pretrained(model_path)
# drop device_map if running on CPU
model = AutoModelForCausalLM.from_pretrained(model_path, device_map=device)
model.eval()
# change input text as desired
chat = [
    { "role": "user", "content": "Please list one IBM Research laboratory located in the United States. You should only output its name and location." },
]
chat = tokenizer.apply_chat_template(chat, tokenize=False, add_generation_prompt=True)
# tokenize the text
input_tokens = tokenizer(chat, return_tensors="pt").to(device)
# generate output tokens
output = model.generate(**input_tokens, 
                        max_new_tokens=100)
# decode output tokens into text
output = tokenizer.batch_decode(output)
# print output
print(output[0])
```

Expected output:
```shell
<|start_of_role|>user<|end_of_role|>Please list one IBM Research laboratory located in the United States. You should only output its name and location.<|end_of_text|>
<|start_of_role|>assistant<|end_of_role|>Almaden Research Center, San Jose, California<|end_of_text|>
```

**Tool-calling:** 
Granite-4.0-1B comes with enhanced tool calling capabilities, enabling seamless integration with external functions and APIs. To define a list of  tools please follow OpenAI's function [definition schema](https://platform.openai.com/docs/guides/function-calling?api-mode=responses#defining-functions). 

This is an example of how to use Granite-4.0-1B model tool-calling ability:

```python
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer

device = "cuda"
model_path = "ibm-granite/granite-4.0-1b"
tokenizer = AutoTokenizer.from_pretrained(model_path)
# drop device_map if running on CPU
model = AutoModelForCausalLM.from_pretrained(model_path, device_map=device)
model.eval()

tools = [
    {
        "type": "function",
        "function": {
            "name": "get_current_weather",
            "description": "Get the current weather for a specified city.",
            "parameters": {
                "type": "object",
                "properties": {
                    "city": {
                        "type": "string",
                        "description": "Name of the city"
                    }
                },
                "required": ["city"]
            }
        }
    }
]

# change input text as desired
chat = [
    { "role": "user", "content": "What's the weather like in Boston right now?" },
]
chat = tokenizer.apply_chat_template(chat, \
                                     tokenize=False, \
                                     tools=tools, \
                                     add_generation_prompt=True)
# tokenize the text
input_tokens = tokenizer(chat, return_tensors="pt").to(device)
# generate output tokens
output = model.generate(**input_tokens, 
                        max_new_tokens=100)
# decode output tokens into text
output = tokenizer.batch_decode(output)
# print output
print(output[0])
```

Expected output:
```shell
<|start_of_role|>system<|end_of_role|>You are a helpful assistant with access to the following tools. You may call one or more tools to assist with the user query.

You are provided with function signatures within <tools></tools> XML tags:
<tools>
{"type": "function", "function": {"name": "get_current_weather", "description": "Get the current weather for a specified city.", "parameters": {"type": "object", "properties": {"city": {"type": "string", "description": "Name of the city"}}, "required": ["city"]}}}
</tools>

For each tool call, return a json object with function name and arguments within <tool_call></tool_call> XML tags:
<tool_call>
{"name": <function-name>, "arguments": <args-json-object>}
</tool_call>. If a tool does not exist in the provided list of tools, notify the user that you do not have the ability to fulfill the request.<|end_of_text|>
<|start_of_role|>user<|end_of_role|>What's the weather like in Boston right now?<|end_of_text|>
<|start_of_role|>assistant<|end_of_role|><tool_call>
{"name": "get_current_weather", "arguments": {"city": "Boston"}}
</tool_call><|end_of_text|>
```

<!-- **Retrieval Augmented Generation:** 
*Coming soon* -->

**Evaluation Results:**

<table>
<thead>
  <tr>
    <th style="text-align:left; background-color: #001d6c; color: white;">Benchmarks</th>
    <th style="text-align:left; background-color: #001d6c; color: white;">Metric</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">350M Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">H 350M Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">1B Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">H 1B Dense</th>
  </tr>
</thead>
  <tbody>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    General Tasks
  </td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MMLU</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">5-shot</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">35.01</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">36.21</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">59.39</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">59.74</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MMLU-Pro</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">5-shot, CoT</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">12.13</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">14.38</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">34.02</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">32.86</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">BBH</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">3-shot, CoT</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">33.07</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">33.28</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">60.37</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">59.68</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">AGI EVAL</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">0-shot, CoT</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">26.22</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">29.61</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">49.22</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">52.44</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">GPQA</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">0-shot, CoT</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">24.11</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">26.12</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">29.91</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">29.69</td>
</tr>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    Alignment Tasks
  </td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">IFEval</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Instruct, Strict</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">61.63</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">67.63</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">80.82</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">82.37</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">IFEval</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Prompt, Strict</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">49.17</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">55.64</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">73.94</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">74.68</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">IFEval</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Average</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">55.4</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">61.63</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">77.38</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">78.53</td>
</tr>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    Math Tasks
  </td>
</tr>      
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">GSM8K</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">8-shot</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">30.71</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">39.27</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">76.35</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">69.83</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">GSM Symbolic</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">8-shot</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">26.76</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">33.7</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">72.3</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">65.72</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">Minerva Math</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">0-shot, CoT</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">13.04</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">5.76</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">45.28</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">49.4</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">DeepMind Math</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">0-shot, CoT</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">8.45</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">6.2</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">34</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">34.98</td>
</tr>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    Code Tasks
  </td>
</tr> 
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">HumanEval</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">39</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">38</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">74</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">73</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">HumanEval+</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">37</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">35</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">69</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">68</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MBPP</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">48</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">49</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">65</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">69</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MBPP+</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">38</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">44</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">57</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">60</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">CRUXEval-O</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">23.75</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">25.5</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">33.13</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">36</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">BigCodeBench</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">11.14</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">11.23</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">30.18</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">29.12</td>
</tr>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    Tool Calling Tasks
  </td>
</tr> 
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">BFCL v3</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;"></td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">39.32</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">43.32</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">54.82</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">50.21</td>
</tr>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    Multilingual Tasks
  </td>
</tr> 
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MULTIPLE</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">pass@1</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">15.99</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">14.31</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">32.24</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">36.11</td>
</tr> 
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MMMLU</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">5-shot</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">28.23</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">27.95</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">45</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">49.43</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">INCLUDE</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">5-shot</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">27.74</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">27.09</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">42.12</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">43.35</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MGSM</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">8-shot</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">14.72</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">16.16</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">37.84</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">27.52</td>
</tr>
<tr>
  <td colspan="6" style="text-align:center; background-color:  #FFFFFF; color: #2D2D2D; font-style:italic;">
    Safety
  </td>
</tr> 
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">SALAD-Bench</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;"></td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">97.12</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">96.55</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">93.44</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">96.4</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">AttaQ</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;"></td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">82.53</td>  
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">81.76</td>
    <td style="text-align:right; background-color: #DAE8FF; color: #2D2D2D;">85.26</td>
    <td style="text-align:right; background-color: #FFFFFF; color: #2D2D2D;">82.85</td>
</tr>
</tbody></table>

<table>
  <caption><b>Multilingual Benchmarks and thr included languages:</b></caption>
<thead>
  <tr>
    <th style="text-align:left; background-color: #001d6c; color: white;">Benchmarks</th>
    <th style="text-align:left; background-color: #001d6c; color: white;"># Langs</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">Languages</th>
  </tr>
</thead>
<tbody>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MMMLU</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">11</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">ar, de, en, es, fr, ja, ko, pt, zh, bn, hi</td>
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">INCLUDE</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">14</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">hi, bn, ta, te, ar, de, es, fr, it, ja, ko, nl, pt, zh</td>
    
</tr>
<tr>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">MGSM</td>
    <td style="text-align:center; background-color: #FFFFFF; color: #2D2D2D;">5</td>
    <td style="text-align:left; background-color: #FFFFFF; color: #2D2D2D;">en, es, fr, ja, zh</td>
</tr>
</tbody>
</table>

**Model Architecture:** 

<!-- Granite-4.0-1B baseline is based on a decoder-only dense transformer architecture. Core components of this architecture are: GQA, Mamba2, MLP with SwiGLU, RMSNorm, and shared input/output embeddings. -->

Granite-4.0-1B baseline is based on a decoder-only dense transformer architecture. Core components of this architecture are: GQA, MLP with SwiGLU, RMSNorm, and shared input/output embeddings.

<table>
<thead>
  <tr>
    <th style="text-align:left; background-color: #001d6c; color: white;">Model</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">350M Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">H 350M Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">1B Dense</th>
    <th style="text-align:center; background-color: #001d6c; color: white;">H 1B Dense</th>
  </tr></thead>
<tbody>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Embedding size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1024</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">768</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">2048</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1536</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of layers</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">28 attention</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">4 attention / 28 Mamba2</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">40 attention</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">4 attention / 36 Mamba2</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Attention head size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">64</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">64</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">128</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of attention heads</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">16</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">12</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">16</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">12</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of KV heads</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">4</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">4</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">4</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">4</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Mamba2 state size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Number of Mamba2 heads</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">48</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">48</td>
  </tr>

  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">MLP / Shared expert hidden size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">2048</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">2048</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">4096</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">4096</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Num. Experts</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Num. active Experts</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Expert hidden size</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">-</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">-</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">MLP activation</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">SwiGLU</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">SwiGLU</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">SwiGLU</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">SwiGLU</td>
  </tr>

  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Sequence length</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">32K</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">32K</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">128K</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">128K</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;">Position embedding</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">RoPE</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">NoPE</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">RoPE</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">NoPE</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;"># Parameters</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">350M</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">340M</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">1.6B</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1.5B</td>
  </tr>
  <tr>
    <td style="text-align:left; background-color: #FFFFFF; color: black;"># Active parameters</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">350M</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">340M</td>
    <td style="text-align:center; background-color: #DAE8FF; color: black;">1.6B</td>
    <td style="text-align:center; background-color: #FFFFFF; color: black;">1.5B</td>
  </tr>
</tbody></table>

**Training Data:** 
Overall, our SFT data is largely comprised of three key sources: (1) publicly available datasets with permissive license, (2) internal synthetic data targeting specific capabilities, and (3) a select set of human-curated data.

**Infrastructure:**
We trained the Granite 4.0 Nano Language Models utilizing an NVIDIA GB200 NVL72 cluster hosted in CoreWeave. Intra-rack communication occurs via the 72-GPU NVLink domain, and a non-blocking, full Fat-Tree NDR 400 Gb/s InfiniBand network provides inter-rack communication. This cluster provides a scalable and efficient infrastructure for training our models over thousands of GPUs.

**Ethical Considerations and Limitations:** 
Granite 4.0 Nano Instruct Models are primarily finetuned using instruction-response pairs mostly in English, but also multilingual data covering multiple languages. Although this model can handle multilingual dialog use cases, its performance might not be similar to English tasks. In such case, introducing a small number of examples (few-shot) can help the model in generating more accurate outputs. While this model has been aligned by keeping safety in consideration, the model may in some cases produce inaccurate, biased, or unsafe responses to user prompts. So we urge the community to use this model with proper safety testing and tuning tailored for their specific tasks.

**Resources**
- ⭐️ Learn about the latest updates with Granite: https://www.ibm.com/granite
- 📄 Get started with tutorials, best practices, and prompt engineering advice: https://www.ibm.com/granite/docs/
- 💡 Learn about the latest Granite learning resources: https://ibm.biz/granite-learning-resources

<!-- ## Citation
```
@misc{granite-models,
  author = {author 1, author2, ...},
  title = {},
  journal = {},
  volume = {},
  year = {2024},
  url = {https://arxiv.org/abs/0000.00000},
}
``` -->