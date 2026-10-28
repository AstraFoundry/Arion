// Verify retained bytes and replay the descriptive report without any model.
import fs from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import {execFileSync} from 'node:child_process';
import assert from 'node:assert/strict';
import {openArchive} from '../../../tools/evaluation/archive/manifest.mjs';
const root=path.resolve(import.meta.dirname,'../../..');
const manifest=path.join(root,'benchmarks/results/asr-hotwords-v1/archive.json');
const archive=await openArchive(manifest,root);
if(archive.manifest.status!=='complete')throw Error('Incomplete archive');
const verified=await archive.verify();
for(const relative of ['tools/evaluation/archive/manifest.mjs','tools/evaluation/src/metrics.mjs','scripts/asr-hotword-bench-command.mjs'])assert.deepEqual(await fs.readFile(path.join(root,relative)),await fs.readFile(path.join(root,'benchmarks/reproduction/asr-hotwords-v1/source-tooling',relative)),'Changed shared report tooling: '+relative);
const temp=await fs.mkdtemp(path.join(os.tmpdir(),'arion-hotwords-replay-'));
try{
 const output=path.join(temp,'summary.json');
 execFileSync(process.execPath,[path.join(root,'benchmarks/reproduction/asr-hotwords-v1/report.mjs'),path.join(root,'benchmarks/runs/asr-hotwords-v1/paired-production'),output,path.join(root,'benchmarks/runs/asr-hotwords-v1/sensitive-isolation')],{stdio:'pipe'});
 const summary=JSON.parse(await fs.readFile(output));
 assert.deepEqual(summary,JSON.parse(await fs.readFile(path.join(root,'benchmarks/results/asr-hotwords-v1/summary.json'))));
 const failed=JSON.parse(await fs.readFile(path.join(root,'benchmarks/runs/asr-hotwords-v1/failed-sandbox-load/run.json')));
 assert.equal(failed.status,'failed');
 assert.equal((await fs.readFile(path.join(root,'benchmarks/runs/asr-hotwords-v1/failed-sandbox-load/raw.jsonl'))).length,0);
 console.log(JSON.stringify({...verified,reportReplay:'passed',failedAttemptPreserved:true,sensitiveContextReset:summary.sensitiveStateIsolation},null,2));
}finally{await fs.rm(temp,{recursive:true,force:true});}
