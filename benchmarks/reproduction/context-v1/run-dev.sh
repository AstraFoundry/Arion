#!/bin/zsh
# Dev-split method iteration, started after the held-out matrix finishes.
set -e
cd /Users/steven/code/Arion
until grep -q '^done' /private/tmp/claude-501/-Users-steven-code-Arion/a8b43b0c-e95e-4acb-a4c8-35ee3edd7bb7/tasks/bl6lhw16t.output 2>/dev/null; do sleep 20; done
C=.workspace/context/dev/context-dev-corpus.json
O=.workspace/benchmarks/context/dev
mkdir -p $O
[ -f $O/ocr.jsonl ] || .workspace/bin/arion-ocr -corpus $C -out $O/ocr.jsonl 2>/dev/null
run() { local model=$1 name=$2; shift 2; caffeinate -i make live-text-bench CORPUS=$C LLM_MODEL=.workspace/ablation-models/$model SCENE=corpus CORRECT=1 VARIANT=$name OUT=$O/$name.jsonl "$@" > $O/$name.log 2>&1; }
run qwen3-1.7b 17-none
run qwen3-1.7b 17-all-prompt CONTEXT=cursor,history,screen SCREEN=accurate OCR=$O/ocr.jsonl CONTEXT_AS=prompt
run qwen3-1.7b 17-all-terms CONTEXT=cursor,history,screen SCREEN=accurate OCR=$O/ocr.jsonl CONTEXT_AS=terms
run qwen3-1.7b 17-all-both CONTEXT=cursor,history,screen SCREEN=accurate OCR=$O/ocr.jsonl CONTEXT_AS=both
run qwen3-4b 4b-all-prompt CONTEXT=cursor,history,screen SCREEN=accurate OCR=$O/ocr.jsonl CONTEXT_AS=prompt
run qwen3-4b 4b-all-terms CONTEXT=cursor,history,screen SCREEN=accurate OCR=$O/ocr.jsonl CONTEXT_AS=terms
echo done
