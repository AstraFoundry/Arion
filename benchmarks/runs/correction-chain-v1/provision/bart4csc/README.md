---
language:
- zh
tags:
- bart
- pytorch
- zh
- Text2Text-Generation
license: apache-2.0
widget:
- text: 少先队员因该为老人让坐
datasets:
- shibing624/CSC
pipeline_tag: text2text-generation
---

# Bart for Chinese Spelling Correction(bart4csc) Model
BART中文拼写纠错模型

`bart4csc-base-chinese` evaluate SIGHAN2015 test data：

Sentence Level: acc:0.6845, precision:0.6984, recall:0.6354, f1:0.6654

case:

|input_text|pred|
|:-- |:--- |
|辰导中引述她的话说：核子间题的解决之道系于克什米尔纷争。|报导中引述她的话说：核子问题的解决之道系于克什米尔纷争。|
|报导并末说明事故发生的原因。|报导并未说明事故发生的原因。|

训练使用了SIGHAN+Wang271K中文纠错数据集，在SIGHAN2015的测试集上达到接近SOTA水平。


## Usage

本项目开源在文本生成项目：[textgen](https://github.com/shibing624/textgen)，可支持Bart模型，通过如下命令调用：

Install package:
```shell
pip install -U textgen
```

```python
from transformers import BertTokenizerFast
from textgen import BartSeq2SeqModel

tokenizer = BertTokenizerFast.from_pretrained('shibing624/bart4csc-base-chinese')
model = BartSeq2SeqModel(
    encoder_type='bart',
    encoder_decoder_type='bart',
    encoder_decoder_name='shibing624/bart4csc-base-chinese',
    tokenizer=tokenizer,
    args={"max_length": 128, "eval_batch_size": 128})
sentences = ["少先队员因该为老人让坐"]
print(model.predict(sentences))
# ['少先队员应该为老人让座']
```


模型文件组成：
```
bart4csc-base-chinese
    ├── config.json
    ├── model_args.json
    ├── pytorch_model.bin
    ├── special_tokens_map.json
    ├── tokenizer_config.json
    ├── spiece.model
    └── vocab.txt
```


### 训练数据集
#### SIGHAN+Wang271K中文纠错数据集


| 数据集 | 语料 | 下载链接 | 压缩包大小 |
| :------- | :--------- | :---------: | :---------: |
| **`SIGHAN+Wang271K中文纠错数据集`** | SIGHAN+Wang271K(27万条) | [百度网盘（密码01b9）](https://pan.baidu.com/s/1BV5tr9eONZCI0wERFvr0gQ)| 106M |
| **`原始SIGHAN数据集`** | SIGHAN13 14 15 | [官方csc.html](http://nlp.ee.ncu.edu.tw/resource/csc.html)| 339K |
| **`原始Wang271K数据集`** | Wang271K | [Automatic-Corpus-Generation dimmywang提供](https://github.com/wdimmy/Automatic-Corpus-Generation/blob/master/corpus/train.sgml)| 93M |


SIGHAN+Wang271K中文纠错数据集，数据格式：
```json
[
    {
        "id": "B2-4029-3",
        "original_text": "晚间会听到嗓音，白天的时候大家都不会太在意，但是在睡觉的时候这嗓音成为大家的恶梦。",
        "wrong_ids": [
            5,
            31
        ],
        "correct_text": "晚间会听到噪音，白天的时候大家都不会太在意，但是在睡觉的时候这噪音成为大家的恶梦。"
    },
]
```


- 如果需要训练Bart模型，请参考[https://github.com/shibing624/textgen/blob/main/examples/seq2seq/training_bartseq2seq_zh_demo.py](https://github.com/shibing624/textgen/blob/main/examples/seq2seq/training_bartseq2seq_zh_demo.py)
- 了解更多纠错模型，请移步：[https://github.com/shibing624/pycorrector](https://github.com/shibing624/pycorrector)

## Citation

```latex
@software{textgen,
  author = {Xu Ming},
  title = {textgen: Implementation of Text Generation models},
  year = {2022},
  url = {https://github.com/shibing624/textgen},
}
```