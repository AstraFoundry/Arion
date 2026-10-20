"""Reuse the frozen comparison executor with an explicit CPU-layer session factory."""
import importlib.util
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
from gguf_session_v2 import Session

spec = importlib.util.spec_from_file_location('cpu_model_comparison_executor', HERE.parent/'rewrite-selector-model-v1/run.py')
executor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(executor)


def cpu_layer_session(*args, gpu_layers, **kwargs):
    if gpu_layers != 99:
        raise ValueError('Changed original comparison executor placement')
    return Session(*args, gpu_layers=0, **kwargs)


if __name__ == '__main__':
    executor.HERE = HERE
    executor.Session = cpu_layer_session
    executor.execute()
