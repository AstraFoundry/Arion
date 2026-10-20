"""Run both complete regression arms without filtering native outputs."""
import argparse,importlib.util,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('private_prior_driver',HERE.parent/'rewrite-current-task-v1/run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
spec=importlib.util.spec_from_file_location('private_span_wrapper',HERE/'wrapper.py');wrapper=importlib.util.module_from_spec(spec);spec.loader.exec_module(wrapper)
runner=driver.runner;runner.HERE=HERE;runner.ARMS={k:('Qwen3.5-2B-Q4_K_M.gguf',True) for k in ['qwen-clean-system','qwen-private-spans']};runner.payload=wrapper.payload;runner.codec=wrapper
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--arm',required=True,choices=runner.ARMS);parser.add_argument('--out',required=True);args=parser.parse_args();sys.argv=[sys.argv[0],'--arm',args.arm,'--out',args.out,'--phase','dev'];runner.main()
