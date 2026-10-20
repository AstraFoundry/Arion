"""Construct four selector-only treatments with identical unverified hypotheses."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
ARMS=['baseline','after','balanced','role']
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
prior=module('screen_prior',HERE.parent/'rewrite-tone-chain-v1/pipeline.py')
after=module('screen_after_data',HERE.parent/'rewrite-selector-preview-v1/preview.py')
comparison=module('screen_comparison_data',HERE/'comparison.py')
selectors={}
for arm in ARMS:
 target=module('screen_payload_'+arm,HERE.parent/'rewrite-bounded-edit-v2/pipeline.py');target.core.HERE=HERE/'templates'/arm;target.core.candidates=prior.new_candidates;target.core.cleaned=lambda sample:sample['speechCleanup'];original=target.core.selection_data
 if arm=='after':target.core.selection_data=lambda p,f=original:after.augment(f,p)
 if arm in ['balanced','role']:target.core.selection_data=lambda p,f=original:comparison.augment(f,p)
 selectors[arm]=target

def candidates_for(arm):
 if arm not in ARMS:raise ValueError('Unknown selector screen arm')
 return prior.new_candidates

def selection_payload(arm,sample,cleanup):
 candidates_for(arm);request,prepared,_=selectors[arm].selection_payload(dict(sample,speechCleanup=cleanup));request['cache_prompt']=True
 return request,prepared

def template_identity(arm,stage,scene):
 candidates_for(arm)
 if stage!='selection':raise ValueError('No formatter executed in selector screen')
 return dict(version='selector-screen-v1-'+arm+'-selection',path='benchmarks/reproduction/rewrite-selector-screen-v1/templates/'+arm+'/prompts.json')

def baseline_projection(request,arm):
 result=json.loads(json.dumps(request,ensure_ascii=False))
 if arm=='role':result['messages'][0]['content']=json.loads((HERE/'templates/baseline/prompts.json').read_text())['selectionSystem']
 for msg in result['messages']:
  if msg['role']=='user':
   data=json.loads(msg['content'])
   for h in data['hypotheses']:
    if arm=='after':h.pop('candidateRewrites')
    if arm in ['balanced','role']:h.pop('candidateComparisons')
   msg['content']=json.dumps(data,ensure_ascii=False,separators=(',',':'))
 return result
