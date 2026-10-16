#!/bin/zsh
# Final held-out context runs with the capped validator; method chosen on dev.
set -e
cd /Users/steven/code/Arion
C=.workspace/context/context-heldout-corpus.json
O=.workspace/benchmarks/context/final
OCR=.workspace/context/ocr.jsonl
mkdir -p $O
run() { local model=$1 name=$2; shift 2; caffeinate -i make live-text-bench CORPUS=$C LLM_MODEL=.workspace/ablation-models/$model SCENE=corpus CORRECT=1 VARIANT=$name OUT=$O/$name.jsonl "$@" > $O/$name.log 2>&1; }
run qwen3-4b 4b-none
run qwen3-4b 4b-terms-cursor CONTEXT=cursor CONTEXT_AS=terms
run qwen3-4b 4b-terms-history CONTEXT=history CONTEXT_AS=terms
run qwen3-4b 4b-terms-screen-oracle CONTEXT=screen SCREEN=oracle CONTEXT_AS=terms
run qwen3-4b 4b-terms-screen-accurate CONTEXT=screen SCREEN=accurate OCR=$OCR CONTEXT_AS=terms
run qwen3-4b 4b-terms-screen-fast CONTEXT=screen SCREEN=fast OCR=$OCR CONTEXT_AS=terms
run qwen3-4b 4b-terms-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR CONTEXT_AS=terms
run qwen3-4b 4b-prompt-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR CONTEXT_AS=prompt
run qwen3-1.7b 17-terms-all-accurate CONTEXT=cursor,history,screen SCREEN=accurate OCR=$OCR CONTEXT_AS=terms
echo done
