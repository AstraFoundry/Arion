"""Reuse the fixed four-arm actual-Go/Metal executor with a contrast-demo adapter."""
import importlib.util
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
import pipeline

spec = importlib.util.spec_from_file_location('contrast_demo_screen_executor', HERE.parent/'rewrite-selector-screen-v1/run.py')
executor = importlib.util.module_from_spec(spec)
spec.loader.exec_module(executor)


if __name__ == '__main__':
    executor.HERE = HERE
    executor.pipeline = pipeline
    executor.execute()
