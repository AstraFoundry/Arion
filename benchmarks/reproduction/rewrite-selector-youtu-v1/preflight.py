"""Run immutable native diagnostics with the owned Youtu artifact freeze."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('youtu_native_executor',HERE.parent/'rewrite-selector-hybrid-v1/preflight.py')
executor=importlib.util.module_from_spec(spec);spec.loader.exec_module(executor)
if __name__=='__main__':
    executor.HERE=HERE
    executor.main()
