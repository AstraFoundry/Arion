// Reuse the frozen production engine driver; labels never enter either driver.
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, writeFileSync, readdirSync, rmSync, copyFileSync, appendFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hash, read } from '../imk-quality-v1/evaluate.mjs';
import { requestsFor } from './requests.mjs';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../../..');
const args=process.argv.slice(2);
if(args.length!==2) throw new Error('usage: prepare.mjs PREPARED_BUILD NEW_OUTPUT');
const build=path.resolve(args[0]),out=path.resolve(args[1]);
const runtime=path.join(root,'apps/desktop/resources/darwin/keyboard'),binary=path.join(build,'driver');
const frozen=read(path.join(build,'metadata.json'));
const corpusFile=path.join(root,'benchmarks/corpora/rime-llm-reranker-v1/corpus.json');
const corpusBytes=readFileSync(corpusFile),corpus=JSON.parse(corpusBytes);
const files=directory=>readdirSync(directory,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(path.join(directory,entry.name)):[path.join(directory,entry.name)]).sort();
const studyFiles=['prepare.mjs','requests.mjs'].map(name=>path.join(path.dirname(fileURLToPath(import.meta.url)),name));
const studyHashes=Object.fromEntries(studyFiles.map(file=>[path.relative(root,file),hash(file)]));
const validate=()=>{
  assert.equal(read(path.join(build,'status.json')).status,'prepared');
  assert.equal(hash(binary),frozen.nativeExecutableSha256);
  for(const [file,digest] of Object.entries({...frozen.sourceHashes,...studyHashes})) assert.equal(hash(path.join(root,file)),digest,file);
  assert.deepEqual(Object.fromEntries(files(runtime).map(file=>[path.relative(runtime,file),hash(file)])),frozen.runtimeFileHashes);
  assert.ok(readFileSync(corpusFile).equals(corpusBytes));
};
validate();
mkdirSync(out,{recursive:false});
for(const name of ['raw','native-requests','profiles','study-sources']) mkdirSync(path.join(out,name));
for(const file of studyFiles) copyFileSync(file,path.join(out,'study-sources',path.basename(file)));
writeFileSync(path.join(out,'corpus.json'),corpusBytes);
const save=(name,value)=>writeFileSync(path.join(out,name),JSON.stringify(value,null,2)+'\n');
const cases=corpus.cases.map(({id,input,context})=>({id,input,context}));
assert.equal(new Set(cases.map(c=>c.id)).size,cases.length);
const warmup={id:'warmup',input:'maomi',context:'院子里有一只'};
const observations=[],requests=[],warmups=[];
save('metadata.json',{...frozen,studyHashes,corpusSha256:hash(corpusFile),design:'current native first-page candidates; typo on; fuzzy off; paired existing context/no-context',limits:['Exposed authored development corpus; not an independent holdout.','No installed host, rendered latency, Windows or ASR contention.']});
save('status.json',{status:'running'});
try {
  for(const fixture of [warmup,...cases]) {
    assert.match(fixture.id,/^[a-z0-9-]+$/);
    const request=`native-requests/${fixture.id}.json`,result=`raw/${fixture.id}.json`,profile=path.join(out,'profiles',fixture.id);
    save(request,{id:fixture.id,input:fixture.input,warmups:['xuetang','heishan'],spelling:{typo:true,fuzzy:0}});
    const child=spawnSync(binary,[path.join(runtime,'data'),profile,path.join(out,request),path.join(out,result)],{encoding:'utf8',timeout:30000});
    appendFileSync(path.join(out,'driver.log'),JSON.stringify({id:fixture.id,status:child.status,error:child.error?.message})+'\n'+(child.stdout??'')+(child.stderr??''));
    assert.equal(child.status,0,fixture.id);
    const raw=read(path.join(out,result)),pair=requestsFor(fixture,raw);
    observations.push({id:fixture.id,request,result,requestSha256:hash(path.join(out,request)),resultSha256:hash(path.join(out,result))});
    if(fixture.id==='warmup') for(let i=0;i<2;++i) warmups.push({...pair[0],session_id:`warmup-${i}`});
    else requests.push(...pair);
    save('observations.json',observations);
    rmSync(profile,{recursive:true,force:true});
  }
  // Deterministic ordering independent of labels, candidates and model output.
  const key=r=>createHash('sha256').update(`42/${r.session_id}`).digest('hex');
  requests.sort((a,b)=>key(a).localeCompare(key(b)));
  writeFileSync(path.join(out,'requests.jsonl'),[...warmups,...requests].map(r=>JSON.stringify(r)+'\n').join(''));
  validate();
  save('status.json',{status:'prepared',nativeProcesses:observations.length,requests:requests.length+warmups.length,requestsSha256:hash(path.join(out,'requests.jsonl'))});
  console.log(`Prepared ${requests.length+warmups.length} reference-free requests from ${observations.length} native processes.`);
} catch(error) { save('status.json',{status:'failed',nativeProcesses:observations.length,error:error.message});throw error; }
