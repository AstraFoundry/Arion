"""Only enable native prefix reuse on the frozen continuous-chain payloads."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('prefix_chain_pipeline',HERE.parent/'rewrite-metal-chain-v1/pipeline.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
candidates=prior.candidates
def selection_payload(sample,cleanup):
 request,prepared,observed=prior.selection_payload(sample,cleanup);request['cache_prompt']=True
 return request,prepared,observed
def format_payload(sample,edited,prepared,choices,edits):
 request,encoding=prior.format_payload(sample,edited,prepared,choices,edits);request['cache_prompt']=True
 return request,encoding
