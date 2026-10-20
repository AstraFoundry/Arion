"""Verify all natural-source/audio lineage offline, without ASR or LLM inference."""
import pathlib,json,hashlib,collections
ROOT=pathlib.Path(__file__).resolve().parents[3];HERE=ROOT/'benchmarks/corpora/rewrite-natural-asr-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def checked(directory=HERE):
 audit=read(directory/'source-audit.json')
 for path,digest in audit['sourceFiles'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed historical evidence '+path)
 for item in audit['upstreamPages']:
  if sha(directory/item['file'])!=item['sha256']:raise ValueError('Changed retained source page')
 inputs=read(directory/'inputs.json');evaluation=read(directory/'evaluator.json');lineage=read(directory/'audio-lineage.json');keys=[r['id'] for r in inputs]
 if len(evaluation)!=120 or len(lineage)!=120 or len(keys)!=120 or len(set(keys))!=120 or set(keys)!={r['id'] for r in evaluation} or set(keys)!={r['id'] for r in lineage}:raise ValueError('Wrong source coverage')
 if collections.Counter(r['language'] for r in inputs)!=dict(zh=60,en=60):raise ValueError('Wrong language coverage')
 if any(set(r)!=set(['id','source','context','scene','language','cohort']) for r in inputs):raise ValueError('Evaluator information leaked into inputs')
 for item in lineage:
  if sha(ROOT/item['audioPath'])!=item['audioSha256']:raise ValueError('Changed original audio')
 corpus=read(ROOT/'benchmarks/corpora/asr-v1/corpus.json');samples={r['id']:r for r in corpus['samples']}
 acoustic=ROOT/'benchmarks/runs/phonetic-asr/phonetic-asr-v1/acoustic/results.jsonl';raw={};line_hashes={}
 for line in acoustic.read_bytes().splitlines(keepends=True):
  record=json.loads(line);raw[record['caseId']]=record['response'];line_hashes[record['caseId']]=hashlib.sha256(line).hexdigest()
 for record in inputs:
  sample=samples[record['id']]
  if record['source']!=raw[record['id']]['asrText'] or record['language']!=sample['language'] or record['context']!={} or record['scene']!='chat' or record['cohort']!='historical-natural-read-speech-regression':raise ValueError('Changed derivative input')
 for record in evaluation:
  sample=samples[record['id']]
  expected=dict(referenceText=sample['reference'],unit=sample['unit'],sourceSpeakerId=sample['speakerId'],sourceSessionId=sample['sessionId'],originalSplit=sample['split'])
  if any(record[k]!=v for k,v in expected.items()):raise ValueError('Changed original evaluator identity/reference')
 for record in lineage:
  sample=samples[record['id']];path=ROOT/'benchmarks/corpora/asr-v1'/sample['audio']['path']
  if record['audioPath']!=str(path.relative_to(ROOT)) or record['audioSha256']!=sample['audio']['sha256'] or record['acousticOriginalLineSha256']!=line_hashes[record['id']]:raise ValueError('Changed audio/ASR lineage')
 if audit['newInference'] is not False:raise ValueError('Source audit is not inference')
 archive=pathlib.Path(__file__).with_name('archive.json')
 if archive.exists():
  for path,digest in read(archive)['files'].items():
   if sha(ROOT/path)!=digest:raise ValueError('Changed source-audit archive')
 return audit
if __name__=='__main__':
 result=checked();print('Verified120 natural read-speech ASR sources and original audio hashes; zero new inference')
