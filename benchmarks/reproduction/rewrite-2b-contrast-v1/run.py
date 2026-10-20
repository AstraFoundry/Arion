"""Reuse the frozen four-scene runner with an explicit <=2B model configuration."""
import importlib.util,pathlib
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('four_scene_runner',HERE.parent/'rewrite-small-prompt-v1/run.py')
prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
prior.HERE=HERE
prior.engine.MODEL=prior.ROOT/'.workspace/rewrite-small-models-v1/Qwen3.5-2B-Q4_K_M.gguf'
# Inherited baseline/compact arm IDs are stable; compact is the contrast package here.
main=prior.main
if __name__=='__main__':main()
