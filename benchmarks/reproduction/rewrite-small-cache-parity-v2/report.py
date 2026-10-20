"""Reuse cache evidence calculations and verify declared token-step configuration."""
import importlib.util,pathlib
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('cache_report_base',HERE.parent/'rewrite-small-cache-v1/report.py')
prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
prior.HERE=HERE;prior.RUN=prior.ROOT/'benchmarks/runs/rewrite-small-cache-parity-v2'
RUN=prior.RUN;ROOT=prior.ROOT;template=prior.template;read=prior.read;sha=prior.sha
base_checked=prior.checked
def checked(directory):
 rows,reviews=base_checked(directory);record=read(directory.parent/'run-environment.json')
 if record['observedBeforeInference'] is not True or record['freezeSha256']!=sha(HERE/'freeze.json') or record['environment']!={'LLAMA_ARG_UBATCH':'1'}:raise ValueError('Wrong pre-inference token-step environment')
 return rows,reviews
prior.checked=checked
main=prior.main
if __name__=='__main__':main()
