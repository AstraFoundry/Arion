// Replay both arms from original raw recognition; labels never enter the driver.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {errorCounts,distribution,NORMALIZATION} from '../../../tools/evaluation/src/metrics.mjs';
const [runDir,output,sensitiveDir]=process.argv.slice(2);
if(!output)throw Error('usage: report.mjs run-directory new-summary.json');
const root=path.resolve(import.meta.dirname,'../../..');
const read=p=>JSON.parse(fs.readFileSync(p,'utf8'));
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const run=read(path.join(runDir,'run.json'));
if(run.status!=='complete'||run.exitCode!==0)throw Error('Incomplete inference run');
if(sha(path.join(runDir,'raw.jsonl'))!==run.rawSha256||sha(path.join(runDir,'stderr.log'))!==run.stderrSha256)throw Error('Changed raw observations');
for(const entry of run.sourceFiles)if(sha(path.join(runDir,'source',entry.path))!==entry.sha256)throw Error(`Changed source: ${entry.path}`);
const evaluation=read(path.join(runDir,'source/benchmarks/corpora/asr-hotwords-v1/evaluation.json'));
const plan=read(path.join(runDir,'source/benchmarks/corpora/asr-hotwords-v1/requests.json'));
const raw=fs.readFileSync(path.join(runDir,'raw.jsonl'),'utf8').trim().split('\n').map(JSON.parse);
if(raw[0]?.type!=='loaded')throw Error('Missing load observation');
const rows=raw.slice(1);
if(rows.length!==plan.length||new Set(rows.map(r=>r.request.id)).size!==plan.length)throw Error('Missing/duplicate observations');
for(let i=0;i<rows.length;i++){
 const row=rows[i],request=plan[i];
 if(row.type!=='result'||row.error||JSON.stringify(row.request)!==JSON.stringify(request)||typeof row.text!=='string'||!Number.isFinite(row.elapsedMs))throw Error('Changed treatment, order or invalid observation');
 if(sha(path.join(root,request.audioPath))!==request.audioSha256)throw Error('Changed original audio');
}
const normalized=s=>s.toLowerCase().replace(/\s/g,'');
const has=(text,term)=>normalized(text).includes(normalized(term));
const perCase=evaluation.cases.map(c=>{
 const observations=rows.filter(r=>r.request.purpose==='measured'&&r.request.caseId===c.caseId);
 if(observations.length!==2||new Set(observations.map(r=>r.request.arm)).size!==2)throw Error('Unpaired case');
 const arms=Object.fromEntries(observations.map(r=>[r.request.arm,{text:r.text,elapsedMs:r.elapsedMs,...errorCounts(c.reference,r.text,c.unit),canonicalHits:c.expectedCanonical.filter(t=>has(r.text,t)),exactCanonicalHits:c.expectedCanonical.filter(t=>r.text.includes(t)),unrelatedInsertions:c.forbiddenInsertions.filter(t=>has(r.text,t)),numericPreserved:(c.numericConcepts??[]).map(n=>n.acceptedSurfaceForms.some(t=>has(r.text,t)))}]));
 return {caseId:c.caseId,origin:c.origin,role:c.role,language:c.language,unit:c.unit,reference:c.reference,hotwords:c.hotwords,expectedCanonical:c.expectedCanonical,forbiddenInsertions:c.forbiddenInsertions,numericConcepts:c.numericConcepts??[],arms,textChanged:arms.empty.text!==arms.canonical.text,errorDelta:arms.canonical.errors-arms.empty.errors,canonicalHitDelta:arms.canonical.canonicalHits.length-arms.empty.canonicalHits.length};
});
const arms={};
for(const arm of ['empty','canonical']){
 const rates={};
 for(const language of ['zh','en','mixed']){
  const cases=perCase.filter(c=>c.language===language&&c.arms[arm].referenceUnits>0);
  const errors=cases.reduce((n,c)=>n+c.arms[arm].errors,0),referenceUnits=cases.reduce((n,c)=>n+c.arms[arm].referenceUnits,0);
  rates[language]={cases:cases.length,errors,referenceUnits,rate:referenceUnits?errors/referenceUnits:null};
 }
 const relevant=perCase.filter(c=>c.role==='relevant'),negative=perCase.filter(c=>c.role==='irrelevant');
 arms[arm]={rates,latencyMs:distribution(perCase.map(c=>c.arms[arm].elapsedMs)),canonicalHits:relevant.reduce((n,c)=>n+c.arms[arm].canonicalHits.length,0),exactCanonicalHits:relevant.reduce((n,c)=>n+c.arms[arm].exactCanonicalHits.length,0),canonicalTargets:relevant.reduce((n,c)=>n+c.expectedCanonical.length,0),negativeCases:negative.length,unrelatedInsertions:negative.reduce((n,c)=>n+c.arms[arm].unrelatedInsertions.length,0),silenceCases:perCase.filter(c=>!c.reference).length,silenceEmissions:perCase.filter(c=>!c.reference&&c.arms[arm].text.trim()).length,numericConcepts:perCase.reduce((n,c)=>n+c.numericConcepts.length,0),numericPreserved:perCase.reduce((n,c)=>n+c.arms[arm].numericPreserved.filter(Boolean).length,0)};
}
const isolation=rows.filter(r=>r.request.purpose==='state-isolation');
if(isolation.length!==4||isolation.map(r=>r.request.arm).join(',')!=='canonical,empty,canonical,empty')throw Error('Missing state isolation sequence');
const initial=perCase.find(c=>c.caseId===isolation[0].request.caseId);
const stateIsolation={sequence:isolation.map(r=>({arm:r.request.arm,hotwords:r.request.hotwords,text:r.text,elapsedMs:r.elapsedMs})),sameContextRepeatsMatch:isolation[0].text===isolation[2].text&&isolation[1].text===isolation[3].text,emptyMatchesPairedBaseline:isolation.filter(r=>r.request.arm==='empty').every(r=>r.text===initial.arms.empty.text),scope:'One resident worker, alternating explicit prompt and empty prompt with identical PCM. This checks observed output isolation, not arbitrary hidden-state equivalence.'};
let sensitiveStateIsolation=null;
if(sensitiveDir){
 const meta=read(path.join(sensitiveDir,'run.json'));
 if(meta.status!=='complete'||sha(path.join(sensitiveDir,'raw.jsonl'))!==meta.rawSha256||sha(path.join(sensitiveDir,'stderr.log'))!==meta.stderrSha256)throw Error('Incomplete/changed sensitive probe');
 for(const entry of meta.sourceFiles)if(sha(path.join(sensitiveDir,'source',entry.path))!==entry.sha256)throw Error('Changed sensitive source');
 const probes=fs.readFileSync(path.join(sensitiveDir,'raw.jsonl'),'utf8').trim().split('\n').map(JSON.parse).slice(1);
 const expected=read(path.join(sensitiveDir,'source/benchmarks/corpora/asr-hotwords-v1/sensitive-isolation-requests.json'));
 if(probes.length!==4||probes.some((p,i)=>p.error||JSON.stringify(p.request)!==JSON.stringify(expected[i])))throw Error('Changed sensitive requests');
 const paired=perCase.find(c=>c.caseId==='codex-a');
 sensitiveStateIsolation={calls:4,sequence:probes.map(p=>({arm:p.request.arm,hotwords:p.request.hotwords,text:p.text,elapsedMs:p.elapsedMs})),contextHasObservableEffect:probes[0].text!==probes[1].text,sameContextRepeatsMatch:probes[0].text===probes[2].text&&probes[1].text===probes[3].text,emptyMatchesPairedBaseline:probes.filter(p=>p.request.arm==='empty').every(p=>p.text===paired.arms.empty.text),canonicalMatchesPairedTreatment:probes.filter(p=>p.request.arm==='canonical').every(p=>p.text===paired.arms.canonical.text),rawSha256:meta.rawSha256};
}
const summary={version:1,status:'complete',design:run.design,stage:run.stage,cases:perCase.length,actualInferenceCalls:rows.length,warmupCalls:rows.filter(r=>r.request.purpose==='warmup').length,measuredCalls:32,stateIsolationCalls:4,loadMs:raw[0].elapsedMs,normalization:NORMALIZATION,arms,changedCases:perCase.filter(c=>c.textChanged).map(c=>c.caseId),improvedErrorCases:perCase.filter(c=>c.errorDelta<0).map(c=>c.caseId),regressedErrorCases:perCase.filter(c=>c.errorDelta>0).map(c=>c.caseId),stateIsolation,sensitiveStateIsolation,canonicalMetric:"Case-insensitive whitespace-normalized substring recall; exact case-sensitive canonical hits are separately reported.",perCase,limitations:evaluation.limitations,rawSha256:run.rawSha256};
fs.writeFileSync(output,JSON.stringify(summary,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({...summary,perCase:undefined,stateIsolation:undefined,sensitiveStateIsolation:undefined},null,2));
