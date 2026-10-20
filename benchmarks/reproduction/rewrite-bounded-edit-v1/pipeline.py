"""Select bounded local spelling edits, then format only the corrected source."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('bounded_source_candidates',HERE/'candidates.py');candidates=importlib.util.module_from_spec(spec);spec.loader.exec_module(candidates)
def read(name):return json.loads((HERE/name).read_text())
def cleaned(sample):
 rows={r['id']:r for r in read('cleanup-observations.json')};row=rows[sample['id']]
 if row['source']!=sample['source']:raise ValueError('Cleanup source mismatch')
 return row['cleanup']
def selection_data(prepared):
 result={k:prepared[k] for k in ['source','context','scene','unknownChineseCharacters']}
 result['hypotheses']=[{k:h[k] for k in ['id','sourceSpan','candidateSpellings','status','kind']} for h in prepared['hypotheses']]
 return result
def selection_payload(sample):
 prompt=read('prompts.json');cleanup=cleaned(sample);prepared=candidates.prepare(cleanup['text'],sample['context'],sample['scene']);schema=candidates.schema(prepared)
 messages=[dict(role='system',content=prompt['selectionSystem'])]
 for example in prompt['selectionExamples']:
  item=candidates.prepare(example['source'],example['context'],example['scene'])
  if example['expectHypotheses']!=len(item['hypotheses']):raise ValueError('Changed declared training-example producer coverage')
  choices={h['id']:('replace' if example['choice']=='replaceUnique' and h['status']=='unique' else 'keep') for h in item['hypotheses']}
  messages.extend([dict(role='user',content=json.dumps(selection_data(item),ensure_ascii=False,separators=(',',':'))),dict(role='assistant',content=json.dumps(choices,separators=(',',':')))])
 messages.append(dict(role='user',content=json.dumps(selection_data(prepared),ensure_ascii=False,separators=(',',':'))))
 return dict(messages=messages,response_format=dict(type='json_object',schema=schema),temperature=0,seed=42,max_tokens=128,cache_prompt=False,stream=False),prepared,cleanup

def format_data(source,encoding):
 types=[]
 for record in encoding['protectedTerms']:
  if record['token'] not in {r['token'] for r in types}:types.append(dict(token=record['token'],kind=record['kind']))
 return dict(source=encoding['encodedSource'],opaqueTokens=types)
def format_payload(sample,edited_source,prepared,selection,edits):
 prompt=read('prompts.json');format_config=prompt['format'][sample['scene']];messages=[dict(role='system',content=prompt['formatSystem']+'\n'+format_config['instruction'])]
 for example in format_config['examples']:
  empty=dict(hypotheses=[]);encoding=candidates.protect(example['source'],empty,{},[]);output=example['output']
  for span in encoding['protectedTerms']:output=output.replace(span['sourceSpan'],span['token'])
  messages.extend([dict(role='user',content=json.dumps(format_data(example['source'],encoding),ensure_ascii=False,separators=(',',':'))),dict(role='assistant',content=output)])
 encoding=candidates.protect(edited_source,prepared,selection,edits)
 messages.append(dict(role='user',content=json.dumps(format_data(edited_source,encoding),ensure_ascii=False,separators=(',',':'))))
 return dict(messages=messages,temperature=0,seed=42,max_tokens=256,cache_prompt=False,stream=False),encoding
