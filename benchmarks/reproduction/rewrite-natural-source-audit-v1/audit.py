"""Verify the complete existing natural read-speech ASR subset without inference."""
import argparse,collections,hashlib,json,pathlib,urllib.request,datetime
ROOT=pathlib.Path(__file__).resolve().parents[3]
CORPUS=ROOT/'benchmarks/corpora/asr-v1/corpus.json'
ACOUSTIC=ROOT/'benchmarks/runs/phonetic-asr/phonetic-asr-v1/acoustic'
VIEW=ROOT/'benchmarks/runs/phonetic-asr/phonetic-asr-v2/completed-17b-view'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def write(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
 corpus=read(CORPUS);acoustic=read(ACOUSTIC/'run.json');view=read(VIEW/'run.json')
 if acoustic['status']!='complete' or view['status']!='complete' or acoustic['resultsHash']!=sha(ACOUSTIC/'results.jsonl') or view['resultsHash']!=sha(VIEW/'results.jsonl'):raise ValueError('Changed/incomplete original evidence')
 if acoustic['corpusHash']!=sha(CORPUS) or view['corpusHash']!=sha(CORPUS) or view['evidenceRunHash']!=sha(ACOUSTIC/'run.json'):raise ValueError('Broken source lineage')
 raw={};raw_lines={}
 for line in (ACOUSTIC/'results.jsonl').read_bytes().splitlines(keepends=True):
  row=json.loads(line);raw[row['caseId']]=row;raw_lines[row['caseId']]=hashlib.sha256(line).hexdigest()
 controls={r['caseId']:r for r in [json.loads(l) for l in (VIEW/'results.jsonl').read_text().splitlines()] if r['variant']=='raw'}
 if len(controls)!=154 or set(controls)!=set(view['sampleIds']):raise ValueError('Wrong historical test control coverage')
 inputs=[];evaluation=[];provenance=[]
 for sample in corpus['samples']:
  if sample['id'] not in controls or not any(tag in sample['tags'] for tag in ['aishell1','librispeech']):continue
  uid=sample['id'];original=raw[uid]['response'];control=controls[uid]
  if not original['ok'] or original['asrText']!=control['response']['text']:raise ValueError('Changed derived ASR source')
  audio=CORPUS.parent/sample['audio']['path']
  if sha(audio)!=sample['audio']['sha256']:raise ValueError('Changed original public audio')
  if sample['split']!='test':raise ValueError('Unexpected original split')
  inputs.append(dict(id=uid,source=original['asrText'],context={},scene='chat',language=sample['language'],cohort='historical-natural-read-speech-regression'))
  evaluation.append(dict(id=uid,referenceText=sample['reference'],unit=sample['unit'],sourceSpeakerId=sample['speakerId'],sourceSessionId=sample['sessionId'],originalSplit=sample['split'],rubricStatus='Published transcript only; independent intended-meaning/style ratings not yet collected. Without corrective evidence, distinguish faithful editing of ASR from repair against spoken reference.'))
  provenance.append(dict(id=uid,audioPath=str(audio.relative_to(ROOT)),audioSha256=sha(audio),durationMs=sample['audio']['durationMs'],acousticOriginalLineSha256=raw_lines[uid],acousticOriginalFile=str((ACOUSTIC/'results.jsonl').relative_to(ROOT)),asrSourceMatchesDerivedView=True))
 counts=collections.Counter(s['language'] for s in inputs)
 if dict(counts)!={'zh':60,'en':60}:raise ValueError('Wrong complete natural subset')
 write(out/'inputs.json',inputs);write(out/'evaluator.json',evaluation);write(out/'audio-lineage.json',provenance)
 source_files=[CORPUS,ACOUSTIC/'run.json',ACOUSTIC/'results.jsonl',VIEW/'run.json',VIEW/'results.jsonl',ROOT/'benchmarks/corpora/asr-v1/provenance.json',pathlib.Path(__file__).resolve()]
 pages=[]
 for slug,url in [('aishell-openslr','https://www.openslr.org/33/'),('librispeech-openslr','https://www.openslr.org/12/')]:
  with urllib.request.urlopen(url,timeout=30) as response:data=response.read()
  path=out/(slug+'.html');path.write_bytes(data);pages.append(dict(url=url,file=path.name,sha256=sha(path),retrievedAt=datetime.datetime.now(datetime.timezone.utc).isoformat()))
 result=dict(kind='source-audit-only',newInference=False,eligibleNaturalSources=len(inputs),languages=dict(counts),audioHashesVerified=len(provenance),asrSourcesVerifiedAgainstOriginal=len(provenance),sourceFiles={str(p.relative_to(ROOT)):sha(p) for p in source_files},upstreamPages=pages,licenseAttribution='Original corpus attribution and hashes preserved. OpenSLR33 lists Apache2 and describes academic use; retain that source-card caveat, not a legal determination. OpenSLR12 lists CC-BY4.0 and LibriVox-derived read speech. See original provenance and stored source pages.',scope='Historical exposed regression, not newly held-out data or spontaneous dictation. Original speakers/sessions/split preserved. Gold references stay evaluator-side. No new ASR, old-Qwen or4B inference. Derived raw control generationMs=0 is a placeholder and must never supply acoustic or rewrite latency. No candidate outputs or independent blind ratings yet. Chinese-only experimental prompts cannot be production defaults without English/mixed-language preservation checks.')
 write(out/'source-audit.json',result);print(json.dumps({k:result[k] for k in ['eligibleNaturalSources','languages','audioHashesVerified','asrSourcesVerifiedAgainstOriginal','newInference']},indent=2))
if __name__=='__main__':main()
