#!/bin/zsh
# Baselines with the shipped live-v2 prompt; sequential so runs don't share CPU.
set -e
cd /Users/steven/code/Arion
M=.workspace/ablation-models/qwen3-1.7b
O=.workspace/benchmarks/scene
caffeinate -i make live-text-bench CORPUS=.workspace/scene/scene-heldout-corpus.json LLM_MODEL=$M SCENE=chat VARIANT=v2-chat OUT=$O/scene-v2-chat.jsonl > $O/scene-v2-chat.log 2>&1
caffeinate -i make live-text-bench CORPUS=.workspace/scene/scene-heldout-corpus.json LLM_MODEL=$M SCENE=auto VARIANT=v2-auto OUT=$O/scene-v2-auto.jsonl > $O/scene-v2-auto.log 2>&1
caffeinate -i make live-text-bench CORPUS=.workspace/live-heldout-corpus.json LLM_MODEL=$M SCENE=corpus VARIANT=v2-heldout OUT=$O/heldout-v2.jsonl > $O/heldout-v2.log 2>&1
echo done
