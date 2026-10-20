"""Short role-aware selector; retain actual candidates and latest formatter."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
prior=module('semantic_v2_prior',HERE.parent/'rewrite-tone-chain-v1/pipeline.py')
selector=module('semantic_v2_selector',HERE.parent/'rewrite-bounded-edit-v2/pipeline.py');selector.core.HERE=HERE;selector.core.candidates=prior.new_candidates;selector.core.cleaned=lambda sample:sample['speechCleanup']
def candidates_for(arm):
 if arm!='candidate':raise ValueError('Unknown semantic-v2 arm')
 return prior.new_candidates
def selection_payload(arm,sample,cleanup):
 candidates_for(arm);request,prepared,_=selector.selection_payload(dict(sample,speechCleanup=cleanup));request['cache_prompt']=True
 return request,prepared
format_payload=prior.format_payload
def template_identity(arm,stage,scene):
 candidates_for(arm)
 if stage=='selection':return dict(version='selector-semantic-v2-selection',path='benchmarks/reproduction/rewrite-selector-semantic-v2/prompts.json')
 if stage=='format':return dict(version='work-scope-v4-'+scene,path='benchmarks/reproduction/rewrite-work-scope-v4/prompts.json')
 raise ValueError('Unknown template stage')
