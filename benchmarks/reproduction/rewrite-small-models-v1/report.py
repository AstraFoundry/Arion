"""Validate complete inference identities and replay raw unfiltered review."""
import argparse,hashlib,json,math,pathlib,sys
import run as study
HERE=study.HERE;ROOT=study.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-small-models-v1'
read=study.read;sha=study.sha
MODELS=['qwen3.5-0.8b','qwen3.5-2b']
def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
def checked(directory):
 frozen=read(HERE/'freeze.json');manifest=read(directory/'manifest.json');review=read(directory/'review.json')
 for path,digest in frozen['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen input: '+path)
 if manifest['status']!='complete' or manifest['generations']!=24 or manifest['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Incomplete or changed inference')
 if manifest['freezeSha256']!=sha(HERE/'freeze.json') or review['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed reviewed freeze')
 identity=manifest['model']
 if identity!=frozen['models'][identity['filename']] or identity['parameters']>2000000000:raise ValueError('Wrong or oversized model')
 if manifest['runtimeFiles']!=frozen['runtimeFiles']:raise ValueError('Wrong runtime')
 expected_command=['--threads','4','--threads-batch','4','--n-gpu-layers','0','--ctx-size','4096','--parallel','1','--reasoning','off','--jinja','--offline','--no-warmup']
 if manifest['command'][-len(expected_command):]!=expected_command:raise ValueError('Wrong CPU generation configuration')
 samples={r['id']:r for r in read(HERE/'cases.json')};rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];judgments={(r['variant'],r['id']):r for r in review['reviews']}
 if len(rows)!=24 or len(judgments)!=24 or len(review['reviews'])!=24 or {r['id'] for r in rows}!=set(samples):raise ValueError('Incomplete or duplicate coverage')
 for row in rows:
  for key,value in samples[row['id']].items():
   if row[key]!=value:raise ValueError('Changed sample')
  if row['variant']!=identity['id'] or row['modelSha256']!=identity['sha256']:raise ValueError('Wrong per-case model')
  template=HERE/f"templates/{row['scene']}.json"
  if row['effectiveModelScene']!=row['scene'] or row['templateSha256']!=sha(template):raise ValueError('Wrong scene/template')
  expected=dict(messages=study.messages(row,read(template)),temperature=0,seed=42,max_tokens=384,cache_prompt=False,stream=False)
  if row['request']!=expected:raise ValueError('Changed actual model request')
  if judgments[(row['variant'],row['id'])]['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed row')
  item=judgments[(row['variant'],row['id'])]
  if not all(type(item[k]) is bool for k in ['proposalFaithful','proposalSceneUseful','repairSucceeded']) or not item['note']:raise ValueError('Incomplete judgment')
 return rows,judgments

def summary(rows,judgments,refs):
 q=lambda values,p:sorted(values)[math.ceil(len(values)*p)-1]
 result=dict(cases=len(rows),rawFaithful=0,rawSceneReady=0,requiredRepairs=0,repairs=0,truncatedOutputs=0,minorSingleBulletOutputs=0)
 for row in rows:
  item=judgments[(row['variant'],row['id'])];result['minorSingleBulletOutputs']+='minor-unnecessary-single-bullet' in item['issues'];repair=refs[row['id']]['mustRepair'];result['rawFaithful']+=item['proposalFaithful'];result['rawSceneReady']+=item['proposalFaithful'] and item['proposalSceneUseful'];result['requiredRepairs']+=repair;result['repairs']+=repair and item['repairSucceeded'];result['truncatedOutputs']+=row['response']['choices'][0]['finish_reason']!='stop'
 result.update(requestP50Ms=q([r['generationMs'] for r in rows],.5),requestP95Ms=q([r['generationMs'] for r in rows],.95),maxObservedResidentRssBytes=max(r['residentRssBytes'] for r in rows),prefillP50Ms=q([r['response']['timings']['prompt_ms'] for r in rows],.5),decodeP50Ms=q([r['response']['timings']['predicted_ms'] for r in rows],.5),maxObservedCachedPromptTokens=max(r['response']['timings']['cache_n'] for r in rows))
 return result

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();refs={r['id']:r for r in read(HERE/'expectations.json')};result=dict(generations=48,reviewedOutputs=48,summary={},productionActivated=False,validatorMode='none; every raw output retained and scored',scope='Text-only GGUF <=2B tensors, CPU4 threads/GPU0, current M4 host, no visual encoder. Same packaging/template and llama.cpp across models. Eight exposed plus sixteen previously ungenerated synthetic probes. No independent natural-data review. Cache disabled; first request includes first-use effects after no-warmup startup. Timings exclude ASR and native insertion; historical ONNX prefix-cached latency is not a causal model comparison. Observed RSS samples are not peak RSS or low-end hardware acceptance.')
 lines=['# Latest small models: complete raw review','','No content validator or fallback hides any output. All source, OCR and generated body personally inspected by the assistant. Eight exposed probes and sixteen previously ungenerated synthetic probes are reported separately. No release adoption.','']
 for model in MODELS:
  directory=RUN/model;rows,judgments=checked(directory);result['summary'][model]=dict(all=summary(rows,judgments,refs),exposed=summary(rows[:8],judgments,refs),fresh=summary(rows[8:],judgments,refs),startupMs=read(directory/'manifest.json')['startupMs'])
  for row in rows:
   item=judgments[(row['variant'],row['id'])]
   template_version=read(HERE/f"templates/{row['scene']}.json")['version']
   lines.extend([f"## {model} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']} (fixture assigned); template: {template_version}",'','**Source**','','```text',row['source'],'```','','**Full OCR context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw output / delivered without validator**','','```text',row['response']['choices'][0]['message']['content'],'```','',item['note'],'',f"Faithful: {item['proposalFaithful']}; scene ready: {item['proposalSceneUseful']}; finish: {row['response']['choices'][0]['finish_reason']}; request: {row['generationMs']:.1f} ms",''])
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);study.write(out/'summary.json',result);(out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
