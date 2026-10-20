"""Change Chinese candidate coverage only; retain selector rules and latest formatter."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
new_candidates=module('tone_chain_candidates',HERE/'candidates.py')
old=module('tone_chain_baseline_selector',HERE.parent/'rewrite-bounded-edit-v2/pipeline.py')
new=module('tone_chain_candidate_selector',HERE.parent/'rewrite-bounded-edit-v2/pipeline.py');new.core.candidates=new_candidates
for selector in [old,new]:selector.core.cleaned=lambda sample:sample['speechCleanup']
formatter=module('tone_chain_routed_formatter',HERE.parent/'rewrite-work-scope-v4/pipeline.py')
def candidates_for(arm):
 if arm=='baseline':return old.candidates
 if arm=='candidate':return new_candidates
 raise ValueError('Unknown tone-chain arm')
def selection_payload(arm,sample,cleanup):
 target=old if arm=='baseline' else new if arm=='candidate' else None
 if target is None:raise ValueError('Unknown tone-chain arm')
 request,prepared,_=target.selection_payload(dict(sample,speechCleanup=cleanup));request['cache_prompt']=True
 return request,prepared
def format_payload(sample,edited,prepared,choices,edits):return formatter.format_payload('candidate',sample,edited,prepared,choices,edits)
