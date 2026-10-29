// Verify archived producer bytes and rebuild every model request independently.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { hash, read } from '../imk-quality-v1/evaluate.mjs';
import { requestsFor } from './requests.mjs';
import { report } from './report.mjs';
import { withEvidence } from './evidence.mjs';
const args=process.argv.slice(2);
if(args.length<1 || args.length>2) throw new Error('usage: verify.mjs CANDIDATE_ARCHIVE [INFERENCE_ARCHIVE]');
withEvidence(args, paths => verify(paths));

function verify(args) {
const archive=path.resolve(args[0]),run=path.join(archive,'run'),manifest=read(path.join(archive,'archive.json'));
const seen=new Set();
for(const entry of manifest.files) {
  assert.ok(!path.isAbsolute(entry.path) && !entry.path.split('/').some(x=>['','.','..'].includes(x)) && !seen.has(entry.path));seen.add(entry.path);
  const file=path.join(archive,entry.path);assert.equal(statSync(file).size,entry.sizeBytes);assert.equal(hash(file),entry.sha256);
}
const meta=read(path.join(run,'metadata.json')),status=read(path.join(run,'status.json'));
assert.equal(status.status,'prepared');assert.equal(hash(path.join(run,'corpus.json')),meta.corpusSha256);
for(const [file,digest] of Object.entries(meta.sourceHashes)) assert.equal(hash(path.join(archive,'sources',file)),digest);
for(const [file,digest] of Object.entries(meta.studyHashes)) assert.equal(hash(path.join(run,'study-sources',path.basename(file))),digest);
const corpus=read(path.join(run,'corpus.json'));
const fixtures=[{id:'warmup',input:'maomi',context:'院子里有一只'},...corpus.cases];
const observations=read(path.join(run,'observations.json'));
assert.equal(observations.length,fixtures.length);assert.equal(status.nativeProcesses,fixtures.length);
const requests=[],warmups=[];
for(const [index,fixture] of fixtures.entries()) {
  const item=observations[index];assert.equal(item.id,fixture.id);
  assert.equal(item.request,`native-requests/${fixture.id}.json`);assert.equal(item.result,`raw/${fixture.id}.json`);
  const request=path.join(run,item.request),result=path.join(run,item.result);
  assert.equal(hash(request),item.requestSha256);assert.equal(hash(result),item.resultSha256);
  assert.deepEqual(read(request),{id:fixture.id,input:fixture.input,warmups:['xuetang','heishan'],spelling:{typo:true,fuzzy:0}});
  const raw=read(result);assert.deepEqual(raw.spelling,{typo:true,fuzzy:0});
  const pair=requestsFor(fixture,raw);
  if(index===0) for(let i=0;i<2;++i) warmups.push({...pair[0],session_id:`warmup-${i}`});
  else requests.push(...pair);
}
const key=r=>createHash('sha256').update(`42/${r.session_id}`).digest('hex');
requests.sort((a,b)=>key(a).localeCompare(key(b)));
const requestFile=path.join(run,'requests.jsonl');assert.equal(hash(requestFile),status.requestsSha256);
assert.equal(status.requests,requests.length+warmups.length);
assert.deepEqual(readFileSync(requestFile,'utf8').trim().split('\n').map(JSON.parse),[...warmups,...requests]);
console.log(`PASS: ${observations.length} actual native snapshots, source hashes and ${status.requests} exact reference-free requests.`);

if(args[1]) {
  const inference=path.resolve(args[1]),record=read(path.join(inference,'archive.json')),members=new Set();
  for(const entry of record.files) {
    assert.ok(!path.isAbsolute(entry.path) && !entry.path.split('/').some(x=>['','.','..'].includes(x)) && !members.has(entry.path));members.add(entry.path);
    const file=path.join(inference,entry.path);assert.equal(statSync(file).size,entry.sizeBytes);assert.equal(hash(file),entry.sha256);
  }
  assert.deepEqual(report(archive,path.join(inference,'run')),read(path.join(inference,'run/report.json')));
  console.log('PASS: original local inference bytes, source identities and exact span-aware report replay.');
}
}
