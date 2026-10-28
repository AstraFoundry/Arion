// Evidence-integrity regressions use retained actual outputs, without inference.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
const root=path.resolve(import.meta.dirname,'../../..');
const original=path.join(root,'benchmarks/runs/asr-hotwords-v1/paired-production');
async function altered(change){
 const temp=await fs.mkdtemp(path.join(os.tmpdir(),'arion-hotwords-integrity-'));
 try{
  const run=path.join(temp,'run');await fs.cp(original,run,{recursive:true});await change(run);
  const result=spawnSync(process.execPath,[path.join(import.meta.dirname,'report.mjs'),run,path.join(temp,'out.json')],{encoding:'utf8'});
  assert.notEqual(result.status,0);
 }finally{await fs.rm(temp,{recursive:true,force:true});}
}
test('incomplete runs cannot produce a completed report',()=>altered(async run=>{
 const p=path.join(run,'run.json'),m=JSON.parse(await fs.readFile(p));m.status='failed';await fs.writeFile(p,JSON.stringify(m));
}));
test('changed original outputs are rejected',()=>altered(async run=>{await fs.appendFile(path.join(run,'raw.jsonl'),'\n');}));
test('duplicate or unpaired observations remain invalid with an updated checksum',()=>altered(async run=>{
 const p=path.join(run,'raw.jsonl');const rows=(await fs.readFile(p,'utf8')).trim().split('\n');rows[2]=rows[1];const bytes=rows.join('\n')+'\n';await fs.writeFile(p,bytes);
 const meta=path.join(run,'run.json'),m=JSON.parse(await fs.readFile(meta));m.rawSha256=crypto.createHash('sha256').update(bytes).digest('hex');await fs.writeFile(meta,JSON.stringify(m));
}));
test('inference plans contain no evaluator labels or reference fields',async()=>{
 const fields=['id','caseId','purpose','arm','audioPath','audioSha256','hotwords'].sort();
 for(const name of ['requests.json','sensitive-isolation-requests.json']){
  const rows=JSON.parse(await fs.readFile(path.join(root,'benchmarks/corpora/asr-hotwords-v1',name)));
  for(const row of rows)assert.deepEqual(Object.keys(row).sort(),fields);
 }
});
