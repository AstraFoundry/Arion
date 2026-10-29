// Score actual span-preserving permutations; annotations remain evaluator-only.
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hash, read } from '../imk-quality-v1/evaluate.mjs';
import { percentile } from '../imk-quality-v1/evaluate.mjs';
const canonical=r=>({...r,candidates:r.candidates.map(c=>({...c,start:c.start??0,end:c.end??0}))});
export function score(requests,observations,corpus,status) {
  assert.equal(status.status,'complete');assert.equal(status.template,'ime-choice-v2');
  assert.equal(status.allowWithoutContext,false);assert.equal(status.count,requests.length);assert.equal(observations.length,requests.length);
  const cases=new Map(corpus.cases.map(c=>[c.id,c]));
  assert.equal(cases.size,corpus.cases.length);
  const groups={context:[], 'no-context':[]},seen=new Set(),warmups=[];
  let errors=0;
  for(const [i,request] of requests.entries()) {
    const o=observations[i];assert.deepEqual(canonical(o.request),canonical(request));
    assert.ok(!seen.has(request.session_id));seen.add(request.session_id);
    assert.ok(Number.isFinite(o.wallMs) && o.wallMs>=0);assert.equal(typeof o.error,'string');
    const cs=request.candidates,ids=cs.map(c=>c.id);assert.equal(new Set(ids).size,ids.length);assert.ok(cs.length);
    const eligible=c=>c.id===cs[0].id || ((cs[0].end??0)>(cs[0].start??0) && (c.start??0)===(cs[0].start??0) && c.end===cs[0].end);
    let order=ids;
    if(o.error) ++errors;
    else {
      const result=o.result;
      for(const key of ['session_id','generation','revision']) assert.equal(result[key],request[key]);
      assert.equal(result.template,'ime-choice-v2');
      order=result.candidates.map(c=>c.id);
      assert.equal(order.length,ids.length);assert.equal(new Set(order).size,ids.length);assert.deepEqual([...order].sort(),[...ids].sort());
      assert.ok(eligible(cs.find(c=>c.id===order[0])),'Promotion must preserve real input coverage');
      assert.deepEqual(order,[order[0],...ids.filter(id=>id!==order[0])],'Only stable promotion is authorized');
      for(const c of result.candidates) if(c.score!==undefined) assert.ok(typeof c.score==='number' && Number.isFinite(c.score));
    }
    const called=(o.completion.chatRequest.messages??[]).length>0;
    if(cs.filter(eligible).length<2) {assert.equal(called,false);assert.deepEqual(order,ids);}

    if(request.session_id.startsWith('warmup-')) { warmups.push({wallMs:o.wallMs,error:o.error,modelCalled:called});continue; }
    const [id,arm,...rest]=request.session_id.split('/');assert.equal(rest.length,0);assert.ok(Object.hasOwn(groups,arm));
    const fixture=cases.get(id);assert.ok(fixture);assert.equal(request.input,fixture.input);
    assert.equal(request.before,arm==='context'?fixture.context:'');
    if(!request.before.trim()) {assert.equal(called,false);assert.deepEqual(order,ids);}
    const byID=new Map(cs.map(c=>[c.id,c.text]));
    const before=cs.map(c=>c.text),after=order.map(id=>byID.get(id));
    const correct=text=>fixture.acceptable.includes(text);
    const rr=words=>{const index=words.findIndex(correct);return index<0?0:1/(index+1);};
    groups[arm].push({id,before:before[0],after:after[0],baselineCorrect:correct(before[0]),rankedCorrect:correct(after[0]),
      rescue:!correct(before[0])&&correct(after[0]),breakage:correct(before[0])&&!correct(after[0]),oracle:before.some(correct),
      eligibleOracle:cs.filter(eligible).some(c=>correct(c.text)),baselineRR:rr(before),rankedRR:rr(after),modelCalled:called,wallMs:o.wallMs,error:o.error});
  }
  assert.equal(errors,status.errors);assert.equal(warmups.length,2);
  const arms={};
  for(const [arm,rows] of Object.entries(groups)) {
    assert.equal(rows.length,cases.size);assert.deepEqual(rows.map(r=>r.id).sort(),[...cases.keys()].sort());
    const calls=rows.filter(r=>r.modelCalled).map(r=>r.wallMs);
    arms[arm]={n:rows.length,...Object.fromEntries(['baselineCorrect','rankedCorrect','rescue','breakage','oracle','eligibleOracle','modelCalled'].map(k=>[k,rows.filter(r=>r[k]).length])),
      errors:rows.filter(r=>r.error).length,baselineMRR:rows.reduce((s,r)=>s+r.baselineRR,0)/rows.length,rankedMRR:rows.reduce((s,r)=>s+r.rankedRR,0)/rows.length,
      modelWallMs:Object.fromEntries([.5,.95,.99].map(q=>[`p${q*100}`,calls.length?percentile(calls,q):null])),cases:rows};
  }
  return {status:'complete',arms,warmups,limitations:['Exposed authored development fixtures; not held-out quality.','Model-call latency excludes rendered UI and installed-host dispatch; no ASR contention.','Errors retain baseline order and remain in denominators.','No personal-history enrollment or model-generated candidates.']};
}
export function report(archive,work) {
  const source=path.join(archive,'run'),requestFile=path.join(work,'requests.jsonl');
  assert.equal(hash(requestFile),hash(path.join(source,'requests.jsonl')));
  const status=read(path.join(work,'inference/status.json'));assert.equal(status.requestsSha256,hash(requestFile));
  const execution=read(path.join(work,'execution.json'));assert.equal(execution.exitCode,0);assert.deepEqual(execution.changedSources,[]);
  const metadata=read(path.join(work,'environment.json'));
  for(const [name,digest] of Object.entries(metadata.sources)) assert.equal(hash(path.join(work,'sources',name)),digest);
  const lines=file=>readFileSync(file,'utf8').trim().split('\n').map(JSON.parse);
  return score(lines(requestFile),lines(path.join(work,'inference/observations.jsonl')),read(path.join(source,'corpus.json')),status);
}
if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  const args=process.argv.slice(2);if(args.length!==3) throw new Error('usage: report.mjs CANDIDATE_ARCHIVE INFERENCE_WORK NEW_REPORT');
  const result=report(args[0],args[1]);writeFileSync(args[2],JSON.stringify(result,null,2)+'\n',{flag:'wx'});
  console.log(JSON.stringify(Object.fromEntries(Object.entries(result.arms).map(([key,{cases,...summary}])=>[key,summary])),null,2));
}
