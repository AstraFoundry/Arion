"""Replay complete default-batch cache timing and raw-output parity evidence."""
import argparse,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_evidence as evidence
ROOT=evidence.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-small-cache-v1';read=evidence.read;sha=evidence.sha
VARIANTS=['forward-uncached','forward-cached','reverse-uncached','reverse-cached']
def template(row):return HERE.parent/f"rewrite-small-models-v1/templates/{row['scene']}.json"
def checked(directory):
 samples=read(HERE/'cases.json')+read(HERE/'warmups.json');rows,reviews=evidence.checked(directory,HERE,{v:samples for v in VARIANTS},template,lambda row:row['variant'].endswith('-cached'),'cache-diagnostic')
 for row in rows:
  if row['order']!=row['variant'].split('-')[0] or row['cacheEnabled']!=row['variant'].endswith('-cached') or row['role']!=('warmup' if row['id'].startswith('cache-warmup-') else 'measured'):raise ValueError('Wrong cache treatment/stratum')
 return rows,reviews

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();directory=RUN/'run';rows,reviews=checked(directory);refs={r['id']:r for r in read(HERE/'expectations.json')};measured=[r for r in rows if r['role']=='measured'];warmups=[r for r in rows if r['role']=='warmup'];lookup={(r['variant'],r['id']):r for r in measured};parity=[];order_parity=[];body=lambda r:r['response']['choices'][0]['message']['content']
 for order in ['forward','reverse']:
  for sample in read(HERE/'cases.json'):
   a=lookup[(order+'-uncached',sample['id'])];b=lookup[(order+'-cached',sample['id'])];parity.append(dict(order=order,id=sample['id'],equal=body(a)==body(b),uncachedRowSha256=evidence.rowhash(a),cachedRowSha256=evidence.rowhash(b)))
 for cache in ['uncached','cached']:
  for sample in read(HERE/'cases.json'):order_parity.append(dict(cache=cache,id=sample['id'],equal=body(lookup[('forward-'+cache,sample['id'])])==body(lookup[('reverse-'+cache,sample['id'])])))
 canaries=['CEDAR-5814','MAPLE-7392'];leaks=[dict(variant=r['variant'],id=r['id']) for r in measured if any(c in body(r) for c in canaries)]
 result=dict(generations=len(rows),reviewedOutputs=len(reviews),measuredGenerations=len(measured),warmupGenerations=len(warmups),summary={v:evidence.summary([r for r in measured if r['variant']==v],reviews,refs) for v in VARIANTS},pooled={name:evidence.summary([r for r in measured if r['cacheEnabled']==flag],reviews,refs) for name,flag in [('uncached',False),('cached',True)]},warmupSummary=evidence.summary(warmups,reviews,{}),rawCacheParity=parity,orderParity=order_parity,syntheticCanaryLeaks=leaks,productionActivated=False,nativeEnvironment=read(directory.parent/'run-environment.json')['environment'] if (directory.parent/'run-environment.json').exists() else {},scope='Eight exposed work/email cases repeated across cache/order; not independent observations or broad quality acceptance. Warmups fully retained/reviewed but excluded from measured denominators. Cached token counts prove tested reuse; differing raw greedy output disproves exact parity even when semantic judgments agree. No low-end host, concurrent ASR, native insertion or general privacy proof.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);evidence.engine.write(out/'summary.json',result);(out/'review.md').write_text(evidence.markdown(rows,reviews,template,'Complete0.8B '+('token-step' if (directory.parent/'run-environment.json').exists() else 'default-batch')+' cache review'));print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
