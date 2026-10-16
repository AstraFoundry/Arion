#!/bin/zsh
# live-v3 runs, sequential; v2 baselines for dev are re-run for same-host timing.
set -e
cd /Users/steven/code/Arion
M=.workspace/ablation-models/qwen3-1.7b
O=.workspace/benchmarks/scene
run() { caffeinate -i make live-text-bench CORPUS=$1 LLM_MODEL=$M SCENE=$2 VARIANT=$3 OUT=$O/$3.jsonl > $O/$3.log 2>&1; }
run .workspace/scene/scene-heldout-corpus.json auto scene-v3-auto
run .workspace/scene/scene-heldout-corpus.json chat scene-v3-chat
run .workspace/live-heldout-corpus.json corpus heldout-v3
run .workspace/ablation-corpus.json corpus dev-v3
echo done
