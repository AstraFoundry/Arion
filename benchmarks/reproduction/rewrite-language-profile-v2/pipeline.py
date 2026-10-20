"""Route trusted policy and author examples by actual source-script presence."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('language_v2_profile',HERE.parent/'rewrite-language-profile-v1/pipeline.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
profile=prior.profile;candidates=prior.candidates
def read():return json.loads((HERE/'prompts.json').read_text())
def route(source):
 p=profile(source)
 return 'han' if p['hasHan'] and not p['hasLatin'] else 'latin' if p['hasLatin'] and not p['hasHan'] else 'mixed'
def format_data(source):return dict(source=source,sourceScripts=profile(source))
def format_payload(sample,edited_source,prepared,choices,edits):
 recipe=read();kind=route(edited_source);scene=recipe['format'][sample['scene']];system=(recipe['formatSystemHan']+'\n'+scene['instructionHan']) if kind=='han' else recipe['formatSystem']+'\n'+scene['instruction']
 system+='\n'+recipe['routeInstructions'][kind];messages=[dict(role='system',content=system)]
 for example in scene['examples']:
  if kind!='mixed' and route(example['source'])!=kind:continue
  if kind=='latin' and profile(example['output'])['hasHan']:raise ValueError('Contradictory Latin demonstration')
  if kind=='han' and profile(example['output'])['hasLatin']:raise ValueError('Contradictory Han demonstration')
  messages += [dict(role='user',content=json.dumps(format_data(example['source']),ensure_ascii=False,separators=(',',':'))),dict(role='assistant',content=example['output'])]
 if len(messages)==1:raise ValueError('Missing declared author examples for source-script route')
 messages.append(dict(role='user',content=json.dumps(format_data(edited_source),ensure_ascii=False,separators=(',',':'))))
 e=candidates.protect(edited_source,prepared,choices,edits);e['policyRoute']=kind
 return dict(messages=messages,temperature=0,seed=42,max_tokens=256,cache_prompt=False,stream=False),e
