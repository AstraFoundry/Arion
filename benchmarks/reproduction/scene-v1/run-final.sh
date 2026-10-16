#!/bin/zsh
# Scene pipeline runs (prompt live-v2, unchanged); sequential to avoid CPU sharing.
set -e
cd /Users/steven/code/Arion
M=.workspace/ablation-models/qwen3-1.7b
O=.workspace/benchmarks/scene
run() { caffeinate -i make live-text-bench CORPUS=$1 LLM_MODEL=$M SCENE=$2 CORRECT=$3 VARIANT=$4 OUT=$O/$4.jsonl > $O/$4.log 2>&1; }
run .workspace/scene/scene-heldout-b-corpus.json chat 1 final-heldoutb-chat
run .workspace/scene/scene-heldout-b-corpus.json auto 1 final-heldoutb-auto
run .workspace/scene/scene-dev-corpus.json chat 1 final-dev-chat
run .workspace/scene/scene-dev-corpus.json auto 1 final-dev-auto
run .workspace/live-heldout-corpus.json corpus 0 final-lrw-heldout
run .workspace/ablation-corpus.json corpus 0 final-lrw-dev
echo done
