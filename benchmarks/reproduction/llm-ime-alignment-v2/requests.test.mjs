import test from 'node:test';
import assert from 'node:assert/strict';
import { requestsFor } from './requests.mjs';
const fixture={id:'fixture',input:'xian',context:'这条',acceptable:['线'],purpose:'EVALUATOR_ONLY'};
const snapshot=()=>({id:'fixture',input:'xian',status:'completed',candidateRangeVersion:1,candidateRangeInspectionMs:0,
  beforeConfirmation:{page:0,candidates:['西安','线','西']},candidateRanges:[
    {index:0,text:'西安',type:'phrase',start:0,end:4},
    {index:1,text:'线',type:'phrase',start:0,end:4},
    {index:2,text:'西',type:'phrase',start:0,end:2}]});
test('real ranges survive different output lengths without leaking labels',()=>{
  const pair=requestsFor(fixture,snapshot());
  assert.equal(pair[0].candidates[1].end,4); assert.equal(pair[0].candidates[2].end,2);
  assert.equal(pair[1].before,''); assert.deepEqual(pair[0].candidates,pair[1].candidates);
  assert.ok(!JSON.stringify(pair).includes('acceptable')); assert.ok(!JSON.stringify(pair).includes('EVALUATOR_ONLY'));
});
test('reject mismatched producer identity, missing spans and invented candidates',()=>{
  for(const mutate of [r=>{r.id='other';},r=>{delete r.candidateRangeVersion;},r=>{r.candidateRanges[1].end=5;},r=>{r.candidateRanges[1].text='invented';},r=>{r.status='failed';}]) {
    const raw=snapshot();mutate(raw);assert.throws(()=>requestsFor(fixture,raw));
  }
});
