import test from 'node:test';
import assert from 'node:assert/strict';
import { score } from './report.mjs';
function fixture() {
  const candidates=[{id:'c0',text:'西安',source:'phrase',start:0,end:4},{id:'c1',text:'线',source:'phrase',start:0,end:4},{id:'c2',text:'西',source:'phrase',start:0,end:2}];
  const requests=['warmup-0','warmup-1','case/context','case/no-context'].map(session_id=>({session_id,generation:1,revision:1,input:'xian',before:session_id.endsWith('no-context')?'':'这条',candidates}));
  const observations=requests.map(request=>({request,result:{session_id:request.session_id,generation:1,revision:1,template:'ime-choice-v2',candidates:(request.before?['c1','c0','c2']:['c0','c1','c2']).map(id=>({id}))},completion:{chatRequest:{messages:request.before?[{role:'system',content:'synthetic test'}]:null}},error:'',wallMs:1}));
  return {requests,observations,corpus:{cases:[{id:'case',input:'xian',context:'这条',acceptable:['线']}]},status:{status:'complete',template:'ime-choice-v2',allowWithoutContext:false,count:4,errors:0}};
}
const run=f=>score(f.requests,f.observations,f.corpus,f.status);
test('different output lengths with equal spans can improve ranking',()=>assert.equal(run(fixture()).arms.context.rescue,1));
test('reject invalid permutations, stale identity, partial promotion and hidden no-context inference',()=>{
  for(const mutate of [
    f=>{f.observations[2].result.candidates=[{id:'c2'},{id:'c0'},{id:'c1'}];},
    f=>{f.observations[2].result.candidates=[{id:'invented'},{id:'c0'},{id:'c2'}];},
    f=>{f.observations[2].result.candidates=[{id:'c1'},{id:'c1'},{id:'c2'}];},
    f=>{f.observations[2].result.revision=2;},
    f=>{f.observations[2].result.candidates[0].score=NaN;},
    f=>{f.observations[3].completion.chatRequest.messages=[{}];},
    f=>{f.status.status='failed';},
    f=>{f.observations.pop();}
  ]) {const f=fixture();mutate(f);assert.throws(()=>run(f));}
});
test('explicit inference errors remain failures in the full denominator',()=>{
  const f=fixture();f.observations[2].error='deadline';f.observations[2].result={};f.status.errors=1;
  const result=run(f);assert.equal(result.arms.context.n,1);assert.equal(result.arms.context.errors,1);assert.equal(result.arms.context.rankedCorrect,0);
});

test('no comparable alternatives cannot hide a model call',()=>{
  const f=fixture();
  for(const r of f.requests) r.candidates=r.candidates.map(c=>({...c,start:0,end:0}));
  for(let i=0;i<f.observations.length;++i) {
    f.observations[i].request=f.requests[i];
    f.observations[i].result.candidates=f.requests[i].candidates.map(({id})=>({id}));
  }
  assert.throws(()=>run(f));
  for(const o of f.observations) o.completion.chatRequest.messages=[];
  assert.equal(run(f).arms.context.modelCalled,0);
});
