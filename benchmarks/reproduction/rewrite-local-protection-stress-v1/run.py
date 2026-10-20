"""Run frozen source-encoding stress cases using the unchanged inference adapter."""
import importlib.util,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
PRIOR=HERE.parent/'rewrite-local-protection-v1'
sys.path.insert(0,str(PRIOR))
spec=importlib.util.spec_from_file_location('local_protection_stress_runner',PRIOR/'run.py');runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
runner.HERE=HERE
runner.ARMS={k:v for k,v in runner.ARMS.items() if k.startswith('qwen-')}
if __name__=='__main__':runner.main()
