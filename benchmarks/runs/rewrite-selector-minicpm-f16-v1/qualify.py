"""Bind personally reviewed F16 native outputs before starting the full comparison."""
from pathlib import Path
import json,hashlib,importlib.util,shutil
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-minicpm-f16-v1';W=R/'.workspace/rewrite-selector-minicpm-f16-v1';D=W/'preflight'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rh(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
assert read(W/'preflight-terminal-status.json')['exitCode']==0
manifest=read(D/'manifest.json');assert manifest['status']=='complete' and manifest['completedResponses']==12
rows=list(map(json.loads,(D/'results.jsonl').read_text().splitlines()));assert len(rows)==12
spec=importlib.util.spec_from_file_location('f16_qualification_report',H/'report.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};labels=[];views=[];viewlabels=[]
for row in rows:
 raw=row['response']['choices'][0]['message']['content'];assert raw=='{"e0":"replace"}' and row['response']['choices'][0]['finish_reason']=='stop'
 correct=row['sourceReference']['id']=='history-0095'
 note='Personally read all12 actual F16 probe responses against complete source/context. '+('Supported load cash -> loadCache is repaired, unlike prior Q4. ' if correct else 'Unsupported read cash -> readCache violates explicit cash/not-cache meaning, unchanged from Q4. ')+('Error persists without schema. ' if row['job']=='free-negative' else '')+'Valid JSON/stop/cache/template are runtime evidence only; no final body or release acceptance.'
 labels.append(dict(id=row['id'],rowSha256=rh(row),protocolConforms=True,probeConforms=correct,note=note))
 view=m.runtime_row(row,refs[row['sourceReference']['arm'],row['sourceReference']['id']]);views.append(view);viewlabels.append(dict(id=view['id'],variant=view['variant'],rowSha256=rh(view),probeConforms=correct,note=note))
parity={str(p):len({r['response']['choices'][0]['message']['content'] for r in rows if r['placement']==p and r['job'] in ['cache-off-first','cache-on-return','cache-off-return']})==1 for p in [99,0]}
save(D/'review.json',dict(resultsSha256=sha(D/'results.jsonl'),method='All12 actual source/context/raw outputs personally inspected;8 supported-repair conforming,4 literal-cash harmful. Repeated2-source diagnostics, not independent release review.',reviews=labels))
save(D/'qualification.json',dict(qualifiedForSelector=parity['99'],productionQualified=False,resultsSha256=sha(D/'results.jsonl'),freezeSha256=sha(H/'freeze.json'),cacheRawParity=parity,reason='Native12 completions stop with valid JSON; actual25/25Metal and0/25CPU layers, pinned template after exact final-LF lexer handling, nonthinking prefix and cache-off/on/off parity.8 task-conforming positive probes,4 harmful literal-cash probes; compatibility only, permits prefrozen202 comparison, no release.'))
V=W/'runtime-view';V.mkdir(exist_ok=False);(V/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in views));save(V/'manifest.json',dict(status='complete',generations=12,resultsSha256=sha(V/'results.jsonl'),rawResultsSha256=sha(D/'results.jsonl'),scope='Derived12-response view, zero additional inference.'));save(V/'review.json',dict(resultsSha256=sha(V/'results.jsonl'),reviews=viewlabels))
m.preflight_checked(D);shutil.copyfile(Path(__file__),H/'original-qualify.py.txt');print('Exact native evidence verified; semantic failure retained.202-request comparison permitted.')
