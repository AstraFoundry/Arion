"""Apply the prefrozen runtime gate without accepting failed semantic choices."""
from pathlib import Path
import hashlib,json
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-minicpm-v1';W=R/'.workspace/rewrite-selector-minicpm-v1';D=W/'preflight'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
m=read(D/'manifest.json');assert m['status']=='complete' and m['attempts']==12 and m['completedResponses']==12 and m['errors']==0 and read(W/'preflight-terminal-status.json')['exitCode']==0
rows=list(map(json.loads,(D/'results.jsonl').read_text().splitlines()));assert m['resultsSha256']==sha(D/'results.jsonl')
for r in rows:
 assert r['response']['choices'][0]['finish_reason']=='stop'
 value=json.loads(r['response']['choices'][0]['message']['content']);assert set(value)=={'e0'} and value['e0'] in ['keep','replace']
for w in m['workers']:
 assert read(D/(w['name']+'-props.json'))['chat_template']==read(H/'sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template']
 log=(D/(w['name']+'-server.log')).read_text()
 assert 'offloaded '+('25/25' if w['placement']==99 else '0/25')+' layers to GPU' in log
 assert 'init: chat template, thinking = 0' in log
 assert '"generation_prompt":"<|im_start|>assistant\\n<think>\\n\\n</think>\\n\\n"' in log
parity={}
for placement in [99,0]:
 selected=[r for r in rows if r['placement']==placement and r['job'] in ['cache-off-first','cache-on-return','cache-off-return']];assert len(selected)==3
 parity[str(placement)]=len({r['response']['choices'][0]['message']['content'] for r in selected})==1
assert parity['99']
(D/'qualification.json').write_text(json.dumps(dict(qualifiedForSelector=True,productionQualified=False,resultsSha256=sha(D/'results.jsonl'),freezeSha256=sha(H/'freeze.json'),cacheRawParity=parity,reason='All12 actual native completions are valid choice JSON with normal stop. Actual pinned template, explicit thinking=0 and empty thinking-block generation prefix observed in logs; Metal25/25 and CPU0/25. Positive cache-off/on/off raw parity holds on both placements. All12 diagnostic semantic choices fail and remain visible. Runtime qualification only permits the prefrozen202-request comparison, not release.'),indent=2)+'\n')
print('Limited runtime qualification passed; all12 semantic task failures retained.')
