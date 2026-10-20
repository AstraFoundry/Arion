"""Replay all direct/encoded requests, raw generations and losslessly restored bodies."""
import argparse,importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_evidence as evidence
spec=importlib.util.spec_from_file_location('protection_runner',HERE/'run.py');runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
ROOT=evidence.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-local-protection-v1';read=evidence.read;sha=evidence.sha

def checked(directory):
 frozen=read(HERE/'freeze.json');manifest=read(directory/'manifest.json');review=read(directory/'review.json');rows=[json.loads(l) for l in (directory/'results.jsonl').read_text().splitlines()];arm=manifest['arm'];phase=manifest['phase']
 for path,digest in frozen['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen input '+path)
 if arm not in runner.ARMS or phase not in ['dev','evaluation'] or manifest['status']!='complete' or manifest['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Wrong/incomplete inference')
 if manifest['model']!=frozen['models'][runner.ARMS[arm][0]] or manifest['runtimeFiles']!=frozen['runtimeFiles']:raise ValueError('Wrong actual model/runtime')
 if any(item['resultsSha256']!=sha(directory/'results.jsonl') for item in [manifest,review]):raise ValueError('Changed reviewed results')
 samples={s['id']:s for s in read(HERE/('development.json' if phase=='dev' else 'fresh.json'))};keys=[(r['variant'],r['id']) for r in rows];expected={(arm,s) for s in samples};judgments={(r['variant'],r['id']):r for r in review['reviews']}
 if len(keys)!=len(expected) or set(keys)!=expected or set(judgments)!=expected or len(review['reviews'])!=len(expected) or manifest['generations']!=len(expected):raise ValueError('Missing/duplicate coverage')
 workers={(w['variant'],w['scene']):w for w in manifest['workers']}
 if len(manifest['workers'])!=4 or set(workers)!={(arm,s) for s in ['work','email','prompt','chat']}:raise ValueError('Wrong worker coverage')
 for worker in workers.values():
  command=worker['command']
  if command[-len(evidence.CPU_ARGUMENTS):]!=evidence.CPU_ARGUMENTS or command[1]!='--model' or pathlib.Path(command[2]).name!=manifest['model']['filename'] or pathlib.Path(command[0]).name!='llama-server' or command[3:7]!=['--host','127.0.0.1','--port','18769']:raise ValueError('Wrong native worker command')
 for row in rows:
  sample=samples[row['id']]
  if any(row[k]!=v for k,v in sample.items()) or row['effectiveModelScene']!=sample['scene']:raise ValueError('Changed original input/scene')
  request,encoding=runner.payload(sample,arm)
  if row['request']!=request or row['encoding']!=encoding:raise ValueError('Changed actual request/encoding')
  if row['templateSha256']!=sha(HERE/f"templates/{sample['scene']}.json") or row['modelSha256']!=manifest['model']['sha256']:raise ValueError('Wrong per-row template/model')
  raw=row['response']['choices'][0]['message']['content'];error=None
  try:body=runner.codec.decode(raw,encoding)
  except ValueError as failure:body=None;error=str(failure)
  if body!=row['renderedBody'] or error!=row['renderError']:raise ValueError('Changed delivered body/protocol error')
  if abs(row['textStageMs']-sum(row[k] for k in ['preparationMs','generationMs','renderMs']))>.001 or min(row[k] for k in ['preparationMs','generationMs','renderMs'])<0:raise ValueError('Wrong timing accounting')
  judgment=judgments[(arm,row['id'])]
  if judgment['rowSha256']!=evidence.rowhash(row) or any(type(judgment[k]) is not bool for k in ['proposalFaithful','proposalSceneUseful','repairSucceeded']) or not judgment['note']:raise ValueError('Changed/incomplete manual review')
 return rows,judgments

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows=[];reviews={};refs={r['id']:r for r in read(HERE/'expectations.json')}
 for arm in runner.ARMS:
  actual,judgments=checked(RUN/arm);rows+=actual;reviews.update(judgments)
 lookup={(r['variant'],r['id']):r for r in rows};controls=[]
 for sample in read(HERE/'development.json'):
  a=lookup[('qwen-direct',sample['id'])];b=lookup[('qwen-protected',sample['id'])]
  controls.append(dict(id=sample['id'],hasProtectedTerms=bool(b['encoding']['protectedTerms']),requestEqual=a['request']==b['request'],bodyEqual=a['renderedBody']==b['renderedBody'],directMs=a['generationMs'],protectedMs=b['generationMs']))
 selection=read(RUN/'qwen-protected/selection.json')
 for arm,hashes in selection['comparisonEvidence'].items():
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if hashes[key]!=sha(RUN/arm/name):raise ValueError('Changed selection comparison')
 if selection['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed selected freeze')
 for filename,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
  if selection[key]!=sha(RUN/'qwen-protected'/filename):raise ValueError('Changed local selected evidence')
 for arm,hashes in selection['stressEvidence'].items():
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if hashes[key]!=sha(ROOT/'benchmarks/runs/rewrite-local-protection-stress-v1'/arm/name):raise ValueError('Changed stress selection evidence')
 result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary={v:evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in runner.ARMS},protocolFailures={v:sum(r['renderError'] is not None for r in rows if r['variant']==v) for v in runner.ARMS},qwenControls=controls,selection=selection,productionActivated=False,scope='Exposed28 synthetic development cases per arm. Common scene prompt edits are a package update, not isolated from the previous study. Within Qwen only local term encoding/instruction/restoration differs; unencoded controls have identical requests. Granite changes model/tokenizer/native template but keeps input messages/decoding/runtime fixed. No semantic output validator or fallback. Protocol errors are retained with null body. Serial timing drift; sampled native RSS excludes Python dictionary/ASR; no native insertion/minimum16GB or independent review acceptance.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);evidence.engine.write(out/'summary.json',result);lines=['# Complete local-protection and non-Qwen screen review','','All raw generated bodies, encoding records, restored bodies and protocol failures personally inspected. No content validator or hidden fallback. Native text and restored body are separate.','']
 for row in rows:
  judgment=reviews[(row['variant'],row['id'])];version=read(HERE/'templates'/(row['scene']+'.json'))['version'];lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}; template: {version}; model: {row['modelSha256']}",'','**Original source**','','```text',row['source'],'```','','**Complete reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Actual encoding**','','```json',json.dumps(row['encoding'],ensure_ascii=False,indent=2),'```','','**Raw native output**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Restored body**','','```text',row['renderedBody'] if row['renderedBody'] is not None else '[null: protocol error]','```','',f"Protocol error: {row['renderError']}",'',judgment['note'],'',f"Faithful: {judgment['proposalFaithful']}; scene useful: {judgment['proposalSceneUseful']}; request {row['generationMs']:.1f}ms",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
