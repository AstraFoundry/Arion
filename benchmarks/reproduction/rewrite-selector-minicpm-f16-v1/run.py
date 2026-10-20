"""Execute immutable native comparison machinery with the owned F16 freeze."""
import importlib.util
import pathlib
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location("minicpm_f16_executor",HERE.parent/'rewrite-selector-model-v1/run.py')
executor=importlib.util.module_from_spec(spec)
spec.loader.exec_module(executor)
if __name__=="__main__":
    executor.HERE=HERE
    executor.execute()
