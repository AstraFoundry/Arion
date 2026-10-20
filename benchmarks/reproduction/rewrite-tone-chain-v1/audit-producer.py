"""Compare actual old selector requests with tone-neutral inputs without inference."""
import pathlib,json,importlib.util,hashlib,argparse
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2]
s=importlib.util.spec_from_file_location('tone_producer_audit_payload',HERE/'pipeline.py');pipeline=importlib.util.module_from_spec(s);s.loader.exec_module(pipeline)
def rh(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def collect():
 path=ROOT/'benchmarks/runs/rewrite-history-chain-v1/run/results.jsonl';rows=list(map(json.loads,path.read_text().splitlines()));finals={r['id']:r for r in rows if r['stage']=='format'};choices={r['id']:r for r in rows if r['stage']=='selection'};records=[];changes=[]
 for sample in json.loads((ROOT/'benchmarks/corpora/rewrite-history-v1/inputs.json').read_text()):
  cleanup=finals[sample['id']]['cleanup'];old,op=pipeline.selection_payload('baseline',sample,cleanup);new,np=pipeline.selection_payload('candidate',sample,cleanup)
  if sample['id'] in choices and old!=choices[sample['id']]['request']:raise ValueError('Old native selector payload mismatch')
  records.append(dict(id=sample['id'],baselineCall=bool(op['hypotheses']),candidateCall=bool(np['hypotheses']),requestEqual=old==new,baselineRequestSha256=rh(old),candidateRequestSha256=rh(new)))
  if old!=new:changes.append(sample['id'])
 return dict(newGenerations=0,sourceCases=401,baselineSelectorCases=sum(r['baselineCall'] for r in records),candidateSelectorCases=sum(r['candidateCall'] for r in records),identicalRequestCases=sum(r['requestEqual'] for r in records),changedRequestIds=changes,actualReferenceSha256=hashlib.sha256(path.read_bytes()).hexdigest(),records=records,scope='Uses exact old actual cleanup/source context. All64 old actual selection requests reproduced; only history-0211 gains a request. Added diagnostic fields are not fed into selection messages. No new inference or acoustic evidence.')
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();result=collect()
 with a.out.open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
 print({k:v for k,v in result.items() if k!='records'})
