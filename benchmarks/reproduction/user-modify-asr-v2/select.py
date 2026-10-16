"""Replay the frozen development-only policy selection; never read test data."""
import argparse,json,pathlib,hashlib
p=argparse.ArgumentParser();p.add_argument('--dev-run',required=True);p.add_argument('--criteria',required=True);p.add_argument('--memory',required=True);p.add_argument('--out',required=True);a=p.parse_args()
r=pathlib.Path(a.dev_run);s=json.loads((r/'summary.json').read_text());criteria=json.loads(pathlib.Path(a.criteria).read_text())
sha=lambda p:hashlib.sha256(pathlib.Path(p).read_bytes()).hexdigest()
if s['status']!='complete':raise ValueError('Incomplete development result')
eligible=[arm for arm in criteria['candidates'] if s['arms'][arm]['newWrongEntityCases']==0 and s['arms'][arm]['baselineCorrectRegressed']==0 and s['arms'][arm]['nonspeechEmissions']==0]
if not eligible:raise ValueError('No candidate qualifies for adoption; any diagnostic test must retain this failure')
rank=lambda arm:(-s['arms'][arm]['primaryEntityHits'],s['arms'][arm]['worsened'],s['arms'][arm]['pipelineMs']['p95'],criteria['candidates'].index(arm))
chosen=min(eligible,key=rank);source=json.loads((r/'run.json').read_text())['sourceHashes']
for path,h in source.items():
 if sha(path)!=h:raise ValueError('Policy source changed before selection')
selection=dict(status='frozen_before_test_inference',selectedArm=chosen,eligibleArms=eligible,selectionCriteriaSha256=sha(a.criteria),devResultsSha256=sha(r/'summary.json'),memorySha256=sha(a.memory),sourceHashes=source,devMetrics={arm:s['arms'][arm] for arm in criteria['candidates']})
with pathlib.Path(a.out).open('x') as f:f.write(json.dumps(selection,indent=2)+'\n')
print(chosen)
