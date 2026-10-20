"""Audit every initial prefix miss without excluding it from full-run metrics."""
import json,pathlib,hashlib,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-metal-prefix-v1/run'
def digest(value):return hashlib.sha256(json.dumps(value,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def quantiles(values):
 values=sorted(values);return {str(q):values[math.ceil(q*len(values))-1] for q in [.5,.95,.99]}
def collect():
 data=(RUN/'results.jsonl').read_bytes();rows=[json.loads(x) for x in data.decode().splitlines()];seen=set();misses=[];miss_ids=set();prefix_count=0
 for index,r in enumerate(rows):
  prefix=digest(r['request']['messages'][:-1]);key=(r['scene'],prefix);first=key not in seen;prefix_count+=first;seen.add(key)
  if r['response']['timings']['cache_n']==0:
   miss_ids.add(r['id']);misses.append(dict(index=index,id=r['id'],stage=r['stage'],scene=r['scene'],policyRoute=r.get('encoding',{}).get('policyRoute'),rowSha256=digest(r),prefixSha256=prefix,firstOccurrenceInWorker=first,requestMs=r['generationMs'],prefillMs=r['response']['timings']['prompt_ms'],decodeMs=r['response']['timings']['predicted_ms']))
 groups={}
 for kind in ['development','natural']:
  for missed in [True,False]:
   subset=[r for r in rows if r['stage']=='format' and r['evaluationSet']==kind and (r['id'] in miss_ids)==missed]
   groups[kind+('-any-prefix-miss' if missed else '-all-stages-hit')]=dict(cases=len(subset),pipelineMs=quantiles([r['pipelineMs'] for r in subset]))
 return dict(kind='post-run-prefix-miss-audit',newGenerations=0,resultsSha256=hashlib.sha256(data).hexdigest(),actualResponses=len(rows),distinctWorkerPrefixes=prefix_count,zeroCacheResponses=len(misses),allMissesAreFirstPrefixOccurrence=all(m['firstOccurrenceInWorker'] for m in misses),misses=misses,groups=groups,scope='Derived from actual response counters. First-occurrence association is not an isolated causal diagnosis; no misses removed from original aggregates. Warm/miss subsets are descriptive and do not replace full quality/latency gates.')
if __name__=='__main__':
 result=collect();print(json.dumps(result,ensure_ascii=False,indent=2))
