"""Record personally read native outputs, then independently check qualification."""
from pathlib import Path
import importlib.util,json,hashlib
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-youtu-v1';W=Path(__file__).resolve().parent;D=W/'preflight'
spec=importlib.util.spec_from_file_location('youtu_owned_report',H/'report.py');report=importlib.util.module_from_spec(spec);spec.loader.exec_module(report)
rows=[json.loads(l) for l in (D/'results.jsonl').read_text().splitlines()];assert len(rows)==12
assert all(r['response']['choices'][0]['message']['content']=='{"e0":"keep"}' for r in rows)
sha=report.sha;rh=report.rowhash
labels=[dict(id=r['id'],rowSha256=rh(r),protocolConforms=True,probeConforms=r['sourceReference']['id']=='history-0279',note='Personally read all12 raw outputs and full final-source/context/hypothesis payloads. All keep. Supported load cash->loadCache correction is missed; explicit cash/not-cache source correctly remains unchanged. Schema-free controls agree. Two repeated exposed sources, not independent release evidence.') for r in rows]
(D/'review.json').write_text(json.dumps(dict(resultsSha256=sha(D/'results.jsonl'),reviews=labels),ensure_ascii=False,indent=2)+'\n')
parity={str(p):len({r['response']['choices'][0]['message']['content'] for r in rows if r['placement']==p and r['job'] in ['cache-off-first','cache-on-return','cache-off-return']})==1 for p in [99,0]}
q=dict(qualifiedForSelector=True,productionQualified=False,resultsSha256=sha(D/'results.jsonl'),freezeSha256=sha(H/'freeze.json'),cacheRawParity=parity,reason='12 normal-stop valid JSON responses, actual33/33Metal and0/33CPU layers, exact upstream/embedded/native template and observed empty-thinking prefix; cache-off/on/off raw parity.4 task-conforming negative controls,8 missed positive repairs. Compatibility only, permits exposed unchanged202 research comparison, never release.')
(D/'qualification.json').write_text(json.dumps(q,ensure_ascii=False,indent=2)+'\n')
refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};v=W/'runtime-view';v.mkdir(exist_ok=False);viewed=[report.runtime_row(r,refs[r['sourceReference']['arm'],r['sourceReference']['id']]) for r in rows]
(v/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in viewed));(v/'manifest.json').write_text(json.dumps(dict(status='complete',resultsSha256=sha(v/'results.jsonl'),rawResultsSha256=sha(D/'results.jsonl')),indent=2)+'\n');(v/'review.json').write_text(json.dumps(dict(resultsSha256=sha(v/'results.jsonl'),reviews=[dict(id=r['id'],variant=r['variant'],rowSha256=rh(r),probeConforms=label['probeConforms'],note=label['note']) for r,label in zip(viewed,labels)]),ensure_ascii=False,indent=2)+'\n')
report.preflight_checked(D)
(W/'qualification-status.json').write_text(json.dumps(dict(status='complete',checked=True,probes=12,protocolConforming=12,taskConforming=4,productionQualified=False),indent=2)+'\n')
print('Exact native evidence qualified for202 comparison; not semantic/release success.')
