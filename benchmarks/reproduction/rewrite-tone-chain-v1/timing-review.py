"""Inspect actual engine/outer timing and both stages of slow completed cases."""
import pathlib,json,hashlib,argparse
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];PATH=ROOT/'benchmarks/runs/rewrite-tone-chain-v1/run/results.jsonl'
def rh(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def collect():
 rows=list(map(json.loads,PATH.read_text().splitlines()));selectors={(r['arm'],r['id']):r for r in rows if r['stage']=='selection'};anomalies=[];cases=[]
 for r in rows:
  t=r['response']['timings']
  if t['prompt_ms']+t['predicted_ms']>r['generationMs']+1:anomalies.append(dict(variant=r['variant'],id=r['id'],rowSha256=rh(r),engineMs=t['prompt_ms']+t['predicted_ms'],outerMs=r['generationMs']))
 for r in rows:
  if r['stage']!='format' or r['cohort']!='historical-synthetic-regression':continue
  sequence=([selectors[(r['arm'],r['id'])]] if (r['arm'],r['id']) in selectors else [])+[r];stages=[dict(stage=x['stage'],rowSha256=rh(x),generationMs=x['generationMs'],cacheTokens=x['response']['timings']['cache_n'],prefillMs=x['response']['timings']['prompt_ms'],decodeMs=x['response']['timings']['predicted_ms'],outputTokens=x['response']['usage']['completion_tokens']) for x in sequence];cases.append(dict(id=r['id'],scene=r['scene'],pipelineMs=r['pipelineMs'],stages=stages,totalPrefillMs=sum(x['prefillMs'] for x in stages),totalDecodeMs=sum(x['decodeMs'] for x in stages)))
 return dict(newGenerations=0,actualResponses=len(rows),resultsSha256=hashlib.sha256(PATH.read_bytes()).hexdigest(),engineExceedsOuterByMoreThan1ms=anomalies,slowestHistoricalSynthetic=sorted(cases,key=lambda x:x['pipelineMs'],reverse=True)[:20],scope='Post-run observational timing breakdown, not a randomized cache-cause experiment. Cache counts and stage times are recorded engine observations, not first-token/native/ASR/concurrent/min16 latency. No historical anomaly is relabeled.')
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();d=collect()
 with a.out.open('x') as f:json.dump(d,f,ensure_ascii=False,indent=2);f.write('\n')
 print('Actual538 response timing bounds checked; flagged',len(d['engineExceedsOuterByMoreThan1ms']))
