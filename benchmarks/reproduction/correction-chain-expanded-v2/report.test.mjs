import test from 'node:test';
import assert from 'node:assert/strict';
import { paired, summarize } from './report.mjs';
test('paired treatment gains and losses remain distinct and silence is excluded',()=>{
 const rows=[
  {caseId:'a',task:'repair',arm:'base',ready:false,fidelity:'degraded'},
  {caseId:'a',task:'repair',arm:'treat',ready:true,fidelity:'preserved'},
  {caseId:'b',task:'copy',arm:'base',ready:true,fidelity:'preserved'},
  {caseId:'b',task:'copy',arm:'treat',ready:false,fidelity:'uncertain'},
  {caseId:'s',task:'silence',arm:'base',ready:false,fidelity:'degraded'},
  {caseId:'s',task:'silence',arm:'treat',ready:true,fidelity:'preserved'}];
 assert.deepEqual(paired(rows,'base','treat','ready'),{spokenCases:2,gains:['a'],losses:['b'],net:0,percentagePointDifference:0});
 assert.deepEqual(paired(rows,'base','treat','fidelity').losses,['b']);
 assert.throws(()=>paired(rows.slice(0,3),'base','treat','ready'),/Missing paired/);
});
test('readability does not erase uncertain fidelity or missing inference',()=>{
 const result=summarize([{fidelity:'uncertain',organization:2,ready:false,unavailable:true,truncated:false,serialEngineEstimateMs:1}]);
 assert.equal(result.uncertain,1);assert.equal(result.preserved,0);assert.equal(result.ready,0);assert.equal(result.unavailable,1);assert.equal(result.strictClean,0);
});

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { observations, root } from './observations.mjs';
import { report } from './report.mjs';
const archivedRun=path.join(root,'benchmarks/runs/correction-chain-expanded-v2/test-run');
test('archived real observations reject changed output bytes before semantic aggregation',t=>{
 const temporary=fs.mkdtempSync(path.join(os.tmpdir(),'arion-chain-'));
 t.after(()=>fs.rmSync(temporary,{recursive:true,force:true}));
 fs.cpSync(archivedRun,temporary,{recursive:true});
 fs.appendFileSync(path.join(temporary,'acoustic.jsonl'),'\n');
 assert.throws(()=>observations(temporary),/Changed output/);
});
test('identical delivered text in the same case cannot receive arm-dependent grades',t=>{
 const temporary=fs.mkdtempSync(path.join(os.tmpdir(),'arion-review-'));
 t.after(()=>fs.rmSync(temporary,{recursive:true,force:true}));
 const ratings=JSON.parse(fs.readFileSync(path.join(root,'benchmarks/results/correction-chain-expanded-v2/ratings.json'),'utf8'));
 const row=ratings.ratings.find(r=>r.caseId==='expanded-silence-0');
 row.fidelity='degraded';row.ready=false;
 const file=path.join(temporary,'ratings.json');fs.writeFileSync(file,JSON.stringify(ratings));
 assert.throws(()=>report(archivedRun,file),/Inconsistent identical-output review/);
});
