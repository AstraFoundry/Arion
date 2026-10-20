"""Keep tone-neutral candidates fixed; revise only candidate selector policy/examples."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
prior=module('semantic_tone_chain_baseline',HERE.parent/'rewrite-tone-chain-v1/pipeline.py')
selector=module('semantic_selector_payload',HERE.parent/'rewrite-bounded-edit-v2/pipeline.py');selector.core.HERE=HERE;selector.core.candidates=prior.new_candidates;selector.core.cleaned=lambda sample:sample['speechCleanup']
candidates_for=lambda arm:prior.new_candidates if arm in ['baseline','candidate'] else (_ for _ in ()).throw(ValueError('Unknown semantic arm'))
def selection_payload(arm,sample,cleanup):
 if arm=='baseline':return prior.selection_payload('candidate',sample,cleanup)
 if arm!='candidate':raise ValueError('Unknown semantic selector arm')
 request,prepared,_=selector.selection_payload(dict(sample,speechCleanup=cleanup));request['cache_prompt']=True
 return request,prepared
format_payload=prior.format_payload
