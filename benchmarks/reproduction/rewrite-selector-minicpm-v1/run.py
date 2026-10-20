"""Run unchanged comparison machinery with owned MiniCPM evidence."""
import importlib.util
import pathlib
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location("minicpm_owned_executor", HERE.parent/'rewrite-selector-model-v1/run.py')
executor=importlib.util.module_from_spec(spec)
spec.loader.exec_module(executor)
if __name__=="__main__":
    executor.HERE=HERE
    executor.execute()
