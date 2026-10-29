// Derive reference-free model requests from actual versioned engine snapshots.
import assert from 'node:assert/strict';
import { candidateCoverage } from '../imk-quality-v1/candidate-ranges.mjs';
export function requestsFor(caseInput, raw) {
  const {id,input,context}=caseInput;
  assert.match(id,/^[a-z0-9-]+$/);
  assert.equal(typeof context,'string');
  assert.equal(raw.id,id); assert.equal(raw.input,input); assert.equal(raw.status,'completed');
  candidateCoverage(raw,{candidateRangeVersion:1},'');
  assert.ok(raw.candidateRanges.length>0);
  const candidates=raw.candidateRanges.map(({index,text,type,start,end})=>({id:`c${index}`,text,source:type,start,end}));
  return ['context','no-context'].map(arm=>({session_id:`${id}/${arm}`,generation:1,revision:1,input,
    before:arm==='context'?context:'',candidates}));
}
