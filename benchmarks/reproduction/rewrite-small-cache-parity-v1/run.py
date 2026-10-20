"""Reuse the frozen cache diagnostic with explicit token-step prefill settings."""
import argparse,importlib.util,json,os,pathlib,sys,datetime
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('prior_cache_runner',HERE.parent/'rewrite-small-cache-v1/run.py')
prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
# The immutable runner resolves inputs and its freeze from this explicit study directory.
prior.HERE=HERE
ROOT=prior.ROOT
ENVIRONMENT={'LLAMA_ARG_BATCH':'1','LLAMA_ARG_UBATCH':'1'}
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args()
 for key,value in ENVIRONMENT.items():os.environ[key]=value
 record=args.out.with_name(args.out.name+'-environment.json');record.parent.mkdir(parents=True,exist_ok=True)
 with record.open('x') as stream:json.dump(dict(observedAt=datetime.datetime.now(datetime.timezone.utc).isoformat(),observedBeforeInference=True,environment={key:os.environ[key] for key in ENVIRONMENT},freezeSha256=prior.engine.sha(HERE/'freeze.json'),reason='Fix logical/physical prefill batch size to one token for cached/uncached numerical-path comparison; native documented environment parameters.'),stream,indent=2)
 prior.main()
if __name__=='__main__':main()
