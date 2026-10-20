"""Execute only the declared hard feasibility probe using the frozen transport."""
import argparse,importlib.util,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('system_probe_driver',HERE.parent/'rewrite-current-task-v1/run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
spec=importlib.util.spec_from_file_location('system_probe_wrapper',HERE/'wrapper.py');wrapper=importlib.util.module_from_spec(spec);spec.loader.exec_module(wrapper)
runner=driver.runner;runner.HERE=HERE;runner.ARMS={'qwen-system':('Qwen3.5-2B-Q4_K_M.gguf',True)};runner.payload=wrapper.payload;runner.codec=wrapper
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--out',required=True);args=parser.parse_args();sys.argv=[sys.argv[0],'--arm','qwen-system','--out',args.out,'--phase','dev'];runner.main()
