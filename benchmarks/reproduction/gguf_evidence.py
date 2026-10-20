"""Validate frozen GGUF requests and complete manually reviewed observations."""
import hashlib,json,pathlib,math
import gguf_rewrite as engine
ROOT=engine.ROOT
read=engine.read;sha=engine.sha
def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
CPU_ARGUMENTS=['--threads','4','--threads-batch','4','--n-gpu-layers','0','--ctx-size','4096','--parallel','1','--reasoning','off','--jinja','--offline','--no-warmup']
def checked(directory,here,groups,template_for,cache_for,phase):
 frozen=read(here/'freeze.json');manifest=read(directory/'manifest.json');reviews=read(directory/'review.json');rows=[json.loads(l) for l in (directory/'results.jsonl').read_text().splitlines()]
 for path,digest in frozen['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen input: '+path)
 if manifest['status']!='complete' or manifest['phase']!=phase or manifest['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Incomplete/changed inference')
 if manifest['freezeSha256']!=sha(here/'freeze.json') or reviews['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed reviewed freeze')
 if manifest['model']!=frozen['model'] or manifest['runtimeFiles']!=frozen['runtimeFiles']:raise ValueError('Wrong model/runtime')
 expected={(variant,sample['id']):sample for variant,samples in groups.items() for sample in samples}
 judgments={(r['variant'],r['id']):r for r in reviews['reviews']}
 if manifest['generations']!=len(expected) or len(rows)!=len(expected) or len(judgments)!=len(expected) or len(reviews['reviews'])!=len(expected):raise ValueError('Incomplete review coverage')
 actual=[(r['variant'],r['id']) for r in rows]
 if len(set(actual))!=len(actual) or set(actual)!=set(expected) or set(judgments)!=set(expected):raise ValueError('Wrong/duplicate variants or cases')
 workers={(w['variant'],w['scene']):w for w in manifest['workers']};expected_workers={(variant,s['scene']) for variant,samples in groups.items() for s in samples}
 if set(workers)!=expected_workers or len(workers)!=len(manifest['workers']):raise ValueError('Wrong worker coverage')
 for worker in workers.values():
  command=worker['command']
  if command[-len(CPU_ARGUMENTS):]!=CPU_ARGUMENTS:raise ValueError('Wrong CPU configuration')
  if command[1]!='--model' or pathlib.Path(command[2]).name!=frozen['model']['filename'] or pathlib.Path(command[0]).name!='llama-server' or command[3:7]!=['--host','127.0.0.1','--port','18769']:raise ValueError('Wrong native command/model/endpoint')
 for row in rows:
  key=(row['variant'],row['id']);sample=expected[key]
  if any(row[k]!=v for k,v in sample.items()) or row['effectiveModelScene']!=sample['scene']:raise ValueError('Changed actual sample/scene')
  template=template_for(row)
  if row['templateSha256']!=sha(template) or row['modelSha256']!=frozen['model']['sha256']:raise ValueError('Wrong per-row template/model')
  expected_request=dict(messages=engine.messages(sample,read(template)),temperature=0,seed=42,max_tokens=384,cache_prompt=cache_for(row),stream=False)
  if row['request']!=expected_request:raise ValueError('Wrong actual request')
  judgment=judgments[key]
  if judgment['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed row')
  if any(type(judgment[k]) is not bool for k in ['proposalFaithful','proposalSceneUseful','repairSucceeded']) or not judgment['note']:raise ValueError('Incomplete judgment')
 return rows,judgments

def summary(rows,judgments,refs):
 q=lambda values,p:sorted(values)[math.ceil(len(values)*p)-1]
 result=dict(cases=len(rows),rawFaithful=0,rawSceneReady=0,requiredRepairs=0,repairs=0,truncated=0)
 for row in rows:
  review=judgments[(row['variant'],row['id'])];repair=refs.get(row['id'],{}).get('mustRepair',False);result['rawFaithful']+=review['proposalFaithful'];result['rawSceneReady']+=review['proposalFaithful'] and review['proposalSceneUseful'];result['requiredRepairs']+=repair;result['repairs']+=repair and review['repairSucceeded'];result['truncated']+=row['response']['choices'][0]['finish_reason']!='stop'
 result.update(requestP50Ms=q([r['generationMs'] for r in rows],.5),requestP95Ms=q([r['generationMs'] for r in rows],.95),prefillP50Ms=q([r['response']['timings']['prompt_ms'] for r in rows],.5),decodeP50Ms=q([r['response']['timings']['predicted_ms'] for r in rows],.5),maxObservedResidentRssBytes=max(r['residentRssBytes'] for r in rows),minCachedTokens=min(r['response']['timings']['cache_n'] for r in rows),maxCachedTokens=max(r['response']['timings']['cache_n'] for r in rows))
 return result

def markdown(rows,judgments,template_for,title):
 lines=['# '+title,'','Every output, including declared warmups, personally inspected. Raw generated body retained without content validation or silent fallback. Scene assignment is fixture supplied; not measured automatic detection.','']
 for row in rows:
  review=judgments[(row['variant'],row['id'])];lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}; template: {read(template_for(row))['version']}; role: {row.get('role','measured')}",'','**Source**','','```text',row['source'],'```','','**Full reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw output / no content validator**','','```text',row['response']['choices'][0]['message']['content'],'```','',review['note'],'',f"Faithful: {review['proposalFaithful']}; scene useful: {review['proposalSceneUseful']}; request: {row['generationMs']:.1f} ms; cached tokens: {row['response']['timings']['cache_n']}",'']
 return '\n'.join(lines)+'\n'
