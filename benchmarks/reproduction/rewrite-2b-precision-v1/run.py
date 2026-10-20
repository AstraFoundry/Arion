"""Execute the declared Q8 artifact only on the full36 exposed regressions."""
import pathlib,importlib.util,argparse,sys
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('precision_prior_driver',HERE.parent/'rewrite-private-spans-v1/run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
spec=importlib.util.spec_from_file_location('precision_wrapper',HERE/'wrapper.py');wrapper=importlib.util.module_from_spec(spec);spec.loader.exec_module(wrapper)
runner=driver.runner;runner.HERE=HERE;runner.ARMS={'qwen-q8-private':('Qwen3.5-2B-Q8_0.gguf',True)};runner.payload=wrapper.payload;runner.codec=wrapper
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--out',required=True);args=parser.parse_args();sys.argv=[sys.argv[0],'--arm','qwen-q8-private','--out',args.out,'--phase','dev'];runner.main()
