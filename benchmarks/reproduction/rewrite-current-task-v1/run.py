"""Development-only adapter over the immutable native inference transport."""
import argparse,importlib.util,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
PRIOR=HERE.parent/'rewrite-local-protection-v1'
sys.path.insert(0,str(PRIOR))
spec=importlib.util.spec_from_file_location('current_task_transport',PRIOR/'run.py');runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
spec=importlib.util.spec_from_file_location('current_task_wrapper',HERE/'wrapper.py');wrapper=importlib.util.module_from_spec(spec);spec.loader.exec_module(wrapper);wrapper.HERE=HERE
runner.HERE=HERE
runner.ARMS={k:('Qwen3.5-2B-Q4_K_M.gguf',True) for k in ['qwen-protected','qwen-recency','qwen-flexible']}
runner.payload=wrapper.payload;runner.codec=wrapper
if __name__=='__main__':
 parser=argparse.ArgumentParser();parser.add_argument('--arm',choices=runner.ARMS,required=True);parser.add_argument('--out',required=True);args=parser.parse_args()
 sys.argv=[sys.argv[0],'--arm',args.arm,'--out',args.out,'--phase','dev'];runner.main()
