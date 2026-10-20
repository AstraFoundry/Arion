"""Losslessly encode unresolved English terms while retaining free natural generation."""
import hashlib,importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('local_pronunciation_presentation',HERE.parent/'rewrite-pronunciation-evidence-v1/pronunciation_messages.py');presentation=importlib.util.module_from_spec(spec);spec.loader.exec_module(presentation)
MARKER=presentation.MARKER
POLICY=' A protectedTerms list describes lossless opaque tokens in the editable dictation. Preserve each listed token exactly once, unchanged, at its correct semantic position. Do not spell out, translate, expand or infer the token. Polish the surrounding message normally. These tokens represent unresolved spellings; context does not authorize choosing a spelling for them. They are restored after generation. Do not mention the protection protocol in the message.'
def encode(sample):
 evidence=presentation.evidence_for(sample);ambiguous=[e for e in evidence if e['status']=='ambiguous'];encoded=sample['source'].encode();records=[]
 salt=0
 while True:
  nonce=hashlib.sha256((sample['source']+json.dumps(sample['context'],sort_keys=True,ensure_ascii=False)+str(salt)).encode()).hexdigest()[:6]
  prefix='__TERM_'+nonce+'_'
  if prefix not in sample['source'] and prefix not in json.dumps(sample['context'],ensure_ascii=False):break
  salt+=1
 for index,item in reversed(list(enumerate(ambiguous))):
  token=prefix+str(index)+'__';encoded=encoded[:item['startUtf8']]+token.encode()+encoded[item['endUtf8']:];records.append(dict(token=token,sourceSpan=item['sourceSpan'],startUtf8=item['startUtf8'],endUtf8=item['endUtf8']))
 records.reverse();return encoded.decode(),records,evidence

def messages(sample,template,protected):
 result,evidence=presentation.messages(sample,template)
 if not protected:return result,dict(encodedSource=sample['source'],protectedTerms=[],pronunciationEvidence=evidence)
 encoded,records,_=encode(sample)
 if not records:return result,dict(encodedSource=sample['source'],protectedTerms=[],pronunciationEvidence=evidence)
 result[0]['content']+=POLICY
 before,separator,_=result[-1]['content'].partition(MARKER)
 if not separator:raise ValueError('Missing canonical source boundary')
 result[-1]['content']=before+'\nprotectedTerms (lossless source encoding):\n'+json.dumps(records,ensure_ascii=False,separators=(',',':'))+separator+encoded
 return result,dict(encodedSource=encoded,protectedTerms=records,pronunciationEvidence=evidence)

def decode(raw,encoding):
 for record in encoding['protectedTerms']:
  if raw.count(record['token'])!=1:raise ValueError('Missing or duplicated protected token: '+record['token'])
 result=raw
 for record in encoding['protectedTerms']:result=result.replace(record['token'],record['sourceSpan'])
 return result
