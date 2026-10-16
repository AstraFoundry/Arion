#!/bin/zsh
# Context ablation: sequential runs, prompt live-v2 (no context) or context-v1.
set -e
cd /Users/steven/code/Arion
C=.workspace/context/context-heldout-corpus.json
O=.workspace/benchmarks/context
OCR=.workspace/context/ocr.jsonl
run() { local model=$1 name=$2; shift 2; caffeinate -i make live-text-bench CORPUS=$C LLM_MODEL=.workspace/ablation-models/$model SCENE=corpus CORRECT=1 VARIANT=$name OUT=$O/$name.jsonl "$@" > $O/$name.log 2>&1; }
run qwen3-1.7b 17-none
run qwen3-1.7b 17-cursor CONTEXT=cursor
run qwen3-1.7b 17-history CONTEXT=history
run qwen3-1.7b 17-screen-oracle CONTEXT=screen SCREEN=oracle
run qwen3-1.7b 17-screen-accurate CONTEXT=screen SCREEN=accurate OCR=$OCR
run qwen3-1.7b 17-screen-fast CONTEXT=screen SCREEN=fast OCR=$OCR
run qwen3-1.7b 17-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR
run qwen3-1.7b 17-all-oracle CONTEXT=cursor,history,screen SCREEN=oracle
run qwen3-4b 4b-none
run qwen3-4b 4b-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR
caffeinate -i make live-text-bench CORPUS=.workspace/live-heldout-corpus.json LLM_MODEL=.workspace/ablation-models/qwen3-1.7b SCENE=corpus VARIANT=lrw-heldout OUT=$O/lrw-heldout.jsonl > $O/lrw-heldout.log 2>&1
echo done
