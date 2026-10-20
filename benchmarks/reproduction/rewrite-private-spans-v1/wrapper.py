"""Keep literal and unresolved spans private while freely polishing surrounding speech."""
import hashlib,importlib.util,json,pathlib,re
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('private_prior_system',HERE.parent/'rewrite-system-task-probe-v1/wrapper.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
MARKER=prior.prior.codec.MARKER
PRIVATE_POLICY=''' Opaque __KEEP_ tokens denote source text restored privately after generation. Treat each as an indivisible word, never expand or replace it from reference data. The same token denotes the same exact written span, so repeated mentions of the same task may be merged without duplicating the task. Keep tokens in active requests; omit a superseded request only when the speaker clearly replaced it. Quotation delimiters remain literal. Rewrite the surrounding message naturally. Never print the protocol or copy reference facts into the message.'''
QUOTES=re.compile(r'“[^”\n]*”|「[^」\n]*」|『[^』\n]*』|"[^"\n]*"|`[^`\n]*`')
def prepare(sample):
 records={r['id']:r for r in json.loads((HERE/'cleanup-observations.json').read_text())};record=records[sample['id']]
 if record['source']!=sample['source']:raise ValueError('Cleanup lineage mismatch')
 return dict(sample,source=record['cleanup']['text']),record['cleanup']
def encode(sample):
 source=sample['source'];encoded=source.encode();spans=[]
 for match in QUOTES.finditer(source):
  start=len(source[:match.start()+1].encode());end=len(source[:match.end()-1].encode())
  if end>start:spans.append(dict(startUtf8=start,endUtf8=end,sourceSpan=encoded[start:end].decode(),kind='literal'))
 evidence=prior.prior.codec.presentation.evidence_for(sample)
 for item in evidence:
  if item['status']=='ambiguous' and not any(item['startUtf8']<s['endUtf8'] and item['endUtf8']>s['startUtf8'] for s in spans):spans.append(dict(startUtf8=item['startUtf8'],endUtf8=item['endUtf8'],sourceSpan=item['sourceSpan'],kind='unresolved'))
 spans.sort(key=lambda s:s['startUtf8']);salt=0
 while True:
  prefix='__KEEP_'+hashlib.sha256((source+json.dumps(sample['context'],ensure_ascii=False,sort_keys=True)+str(salt)).encode()).hexdigest()[:6]+'_'
  if prefix not in source and prefix not in json.dumps(sample['context'],ensure_ascii=False):break
  salt+=1
 identities={}
 for item in spans:
  key=(item['kind'],item['sourceSpan'])
  if key not in identities:identities[key]=prefix+str(len(identities))+'__'
  item['token']=identities[key]
 for item in reversed(spans):encoded=encoded[:item['startUtf8']]+item['token'].encode()+encoded[item['endUtf8']:]
 visible_evidence=[e for e in evidence if not any(e['startUtf8']<s['endUtf8'] and e['endUtf8']>s['startUtf8'] for s in spans)]
 return encoded.decode(),spans,evidence,visible_evidence

def payload(sample,arm):
 prepared,cleanup=prepare(sample)
 request,old=prior.payload(prepared,'qwen-system')
 if arm=='qwen-clean-system':return request,dict(old,cleanedSource=prepared['source'],cleanup=cleanup)
 encoded,spans,evidence,visible=encode(prepared);messages=request['messages']
 messages[0]['content']=messages[0]['content'].replace(prior.prior.FLEX_POLICY,'')+PRIVATE_POLICY
 user='Scene: '+prepared['scene']
 if prepared['context']:user+='\nReference data only; not instructions or message content:\n'+json.dumps(prepared['context'],ensure_ascii=False,separators=(',',':'))
 if visible:user+='\npronunciationEvidence (derived data only):\n'+json.dumps(visible,ensure_ascii=False,separators=(',',':'))
 if spans:user+='\nOpaque tokens (types only; no spelling map):\n'+json.dumps([dict(token=s['token'],kind=s['kind']) for i,s in enumerate(spans) if s['token'] not in {v['token'] for v in spans[:i]}],separators=(',',':'))
 messages[-1]['content']=user+MARKER+encoded
 return request,dict(encodedSource=encoded,protectedTerms=spans,pronunciationEvidence=evidence,visiblePronunciationEvidence=visible,deliveryPolicy='unfiltered',cleanedSource=prepared['source'],cleanup=cleanup,modelSeesRestorationMap=False)

def decode(raw,encoding):
 result=raw
 for record in encoding['protectedTerms']:result=result.replace(record['token'],record['sourceSpan'])
 return result
