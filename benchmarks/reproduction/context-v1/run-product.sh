#!/bin/zsh
set -e
cd /Users/steven/code/Arion
C=.workspace/context/context-heldout-corpus.json
O=.workspace/benchmarks/context/final
OCR=.workspace/context/ocr.jsonl
run() { local model=$1 name=$2; shift 2; caffeinate -i make live-text-bench CORPUS=$C LLM_MODEL=.workspace/ablation-models/$model SCENE=corpus CORRECT=1 VARIANT=$name OUT=$O/$name.jsonl "$@" > $O/$name.log 2>&1; }
run qwen3-4b 4b-product-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR CONTEXT_AS=terms
run qwen3-1.7b 17-product-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR CONTEXT_AS=terms
echo done
