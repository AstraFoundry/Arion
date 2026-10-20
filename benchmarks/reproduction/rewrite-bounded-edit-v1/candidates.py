"""Build source-relative spelling hypotheses from fixed text-derived pronunciations."""
import pathlib,json,re,importlib.util,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
spec=importlib.util.spec_from_file_location('bounded_english_phonetics',ROOT/'benchmarks/reproduction/rewrite-pronunciation-evidence-v1/pronunciation.py');english=importlib.util.module_from_spec(spec);spec.loader.exec_module(english)
READINGS=json.loads((ROOT/'benchmarks/artifacts/pypinyin-character-readings-v1/readings.json').read_text());LEXICON=None
HAN=re.compile(r'[\u3400-\u9fff]+');QUOTED=english.QUOTED
NUMBER=re.compile(r'[0-9零〇一二三四五六七八九十百千万亿两点]+(?:元|块|秒|天|小时|分钟|米|户|个|次|年|月|日)')
CODE=re.compile(r'(?<![A-Za-z0-9_])(?:/[A-Za-z0-9_./-]+|[A-Za-z]+(?:-[A-Za-z0-9]+)+|[A-Za-z0-9_-]+\.[A-Za-z][A-Za-z0-9]{0,8}|[a-z]+(?:[A-Z][A-Za-z0-9]*)+|[A-Z]+[0-9]+)(?![A-Za-z0-9_])')
def utf8(source,index):return len(source[:index].encode())
def overlap(a,b):return a[0]<b[1] and a[1]>b[0]
def screen_text(context):return '\n'.join(str(v) for v in context.values())
def han_windows(text):
 for match in HAN.finditer(text):
  for length in range(2,7):
   for start in range(match.start(),match.end()-length+1):yield start,start+length,text[start:start+length]
def same_reading(a,b):return len(a)==len(b) and all(set(READINGS.get(x,())).intersection(READINGS.get(y,())) for x,y in zip(a,b))
def prepare(source,context,scene):
 global LEXICON
 if LEXICON is None:LEXICON=english.Lexicon()
 screen=screen_text(context);literals=[m.span() for m in QUOTED.finditer(source)];numbers=[m.span() for m in NUMBER.finditer(source)];hypotheses=[]
 for item in LEXICON.align(source,screen,scene):
  spellings=item['candidateSpellings'];status=item['status']
  if status=='unknown_pronunciation':
   collapsed=''.join(item['sourceSpan'].split()).lower();spellings=sorted(set(w for w in english.IDENTIFIER.findall(screen) if w.lower()==collapsed))
   if spellings:status='unique' if len(spellings)==1 else 'ambiguous'
  if status in ['unique','ambiguous']:
   hypotheses.append(dict(startUtf8=item['startUtf8'],endUtf8=item['endUtf8'],sourceSpan=item['sourceSpan'],candidateSpellings=spellings,status=status,kind='english-identifier'))
 targets=sorted(set(word for _,_,word in han_windows(screen)))
 for start,end,word in han_windows(source):
  if any(overlap((start,end),span) for span in literals+numbers) or word in screen:continue
  matches=[target for target in targets if same_reading(word,target)]
  if matches:hypotheses.append(dict(startUtf8=utf8(source,start),endUtf8=utf8(source,end),sourceSpan=word,candidateSpellings=matches,status='unique' if len(matches)==1 else 'ambiguous',kind='chinese-character-readings'))
 # Prefer maximal nonoverlapping hypotheses; never use case IDs or evaluator labels.
 chosen=[]
 for item in sorted(hypotheses,key=lambda h:(-(h['endUtf8']-h['startUtf8']),h['startUtf8'])):
  if not any(overlap((item['startUtf8'],item['endUtf8']),(x['startUtf8'],x['endUtf8'])) for x in chosen):chosen.append(item)
 chosen.sort(key=lambda h:h['startUtf8'])
 for index,item in enumerate(chosen):item['id']='e'+str(index)
 unknown=sorted(set(c for c in source+screen if HAN.fullmatch(c) and c not in READINGS))
 return dict(source=source,context=context,scene=scene,hypotheses=chosen,unknownChineseCharacters=unknown,scope='Fixed character-level heteronym/tone and CMU text evidence, not acoustic truth. Chinese2..6-character maximal spans; exact source spellings/quoted/numeric spans excluded. Unknown English direct letter/boundary matches are limited hypotheses. Full OCR stays in classifier input.')
def schema(prepared):
 fields={h['id']:dict(type='string',enum=['keep','replace'] if h['status']=='unique' else ['keep']) for h in prepared['hypotheses']}
 return dict(type='object',properties=fields,required=list(fields),additionalProperties=False)
def apply(raw,prepared):
 decisions=json.loads(raw);expected={h['id'] for h in prepared['hypotheses']}
 if not isinstance(decisions,dict) or set(decisions)!=expected:raise ValueError('Wrong edit-selection protocol coverage')
 source=prepared['source'].encode();edits=[]
 for h in prepared['hypotheses']:
  allowed=schema(prepared)['properties'][h['id']]['enum']
  if decisions[h['id']] not in allowed:raise ValueError('Wrong edit-selection value')
  if decisions[h['id']]=='replace':edits.append(dict(startUtf8=h['startUtf8'],endUtf8=h['endUtf8'],original=h['sourceSpan'],replacement=h['candidateSpellings'][0],hypothesisId=h['id']))
 for edit in reversed(edits):
  if source[edit['startUtf8']:edit['endUtf8']].decode()!=edit['original']:raise ValueError('Source edit identity mismatch')
  source=source[:edit['startUtf8']]+edit['replacement'].encode()+source[edit['endUtf8']:]
 return source.decode(),edits

def protect(source,prepared,decisions,edits):
 spans=[]
 for match in QUOTED.finditer(source):
  start,end=match.span();spans.append(dict(startUtf8=utf8(source,start),endUtf8=utf8(source,end),sourceSpan=match.group(),kind='literal'))
 for match in CODE.finditer(source):
  a,b=utf8(source,match.start()),utf8(source,match.end())
  if not any(overlap((a,b),(s['startUtf8'],s['endUtf8'])) for s in spans):spans.append(dict(startUtf8=a,endUtf8=b,sourceSpan=match.group(),kind='code'))
 for h in prepared['hypotheses']:
  if decisions[h['id']]!='keep':continue
  for match in re.finditer(re.escape(h['sourceSpan']),source):
   a,b=utf8(source,match.start()),utf8(source,match.end())
   if not any(overlap((a,b),(s['startUtf8'],s['endUtf8'])) for s in spans):spans.append(dict(startUtf8=a,endUtf8=b,sourceSpan=match.group(),kind='unresolved'))
 for edit in edits:
  for match in re.finditer(re.escape(edit['replacement']),source):
   a,b=utf8(source,match.start()),utf8(source,match.end())
   if not any(overlap((a,b),(s['startUtf8'],s['endUtf8'])) for s in spans):spans.append(dict(startUtf8=a,endUtf8=b,sourceSpan=match.group(),kind='committed-spelling'))
 spans.sort(key=lambda s:s['startUtf8']);salt=0
 while True:
  prefix='__KEEP_'+hashlib.sha256((source+str(salt)).encode()).hexdigest()[:6]+'_'
  if prefix not in source:break
  salt+=1
 identities={}
 for span in spans:
  key=(span['kind'],span['sourceSpan'])
  if key not in identities:identities[key]=prefix+str(len(identities))+'__'
  span['token']=identities[key]
 encoded=source.encode()
 for span in reversed(spans):encoded=encoded[:span['startUtf8']]+span['token'].encode()+encoded[span['endUtf8']:]
 return dict(encodedSource=encoded.decode(),protectedTerms=spans,deliveryPolicy='unfiltered')
def restore(raw,encoding):
 result=raw
 for span in encoding['protectedTerms']:result=result.replace(span['token'],span['sourceSpan'])
 return result
