// Replay descriptive native ASR results from retained outputs; no inference.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {errorCounts,distribution,NORMALIZATION} from '../../../tools/evaluation/src/metrics.mjs';
const [corpusFile, runsRoot, output]=process.argv.slice(2);
if(!output) throw Error('usage: report.mjs corpus.json runs-root new-summary.json');
const read=p=>JSON.parse(fs.readFileSync(p,'utf8'));
const lines=p=>fs.readFileSync(p,'utf8').trim().split('\n').map(JSON.parse);
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const corpus=read(corpusFile),samples=corpus.samples.filter(s=>s.split==='dev');
const summary={normalization:NORMALIZATION,split:'dev',cases:samples.length,corpusSha256:sha(corpusFile),arms:{}};
for(const name of ['compute-v2','packaged-offline-v1','packaged-offline-final','pipeline-v2']){
 const dir=path.join(runsRoot,name),run=read(path.join(dir,'run.json'));
 if(run.status!=='complete') throw Error(`Incomplete run: ${name}`);
 for(const [file,key] of [['results.jsonl','resultsSha256'],['warmups.jsonl','warmupsSha256']]) if(sha(path.join(dir,file))!==run[key]) throw Error(`Changed results: ${name}`);
 const rows=lines(path.join(dir,'results.jsonl'));
 if(rows.length!==samples.length||new Set(rows.map(r=>r.caseId)).size!==samples.length) throw Error('Missing/duplicate cases');
 const perCase=samples.map(s=>{const row=rows.find(r=>r.caseId===s.id);if(!row)throw Error('Missing input');const text=name==='pipeline-v2'?row.response.finalText:row.text;if(name!=='pipeline-v2'&&row.audioSha256!==s.audio.sha256)throw Error('Changed audio');return {caseId:s.id,language:s.language,text,elapsedMs:name==='pipeline-v2'?row.metrics.stopToReviewMs:row.elapsedMs,...errorCounts(s.reference,text,s.unit)}});
 const rates={};for(const language of ['zh','en','mixed']){const rows=perCase.filter(r=>r.language===language&&r.referenceUnits>0);const errors=rows.reduce((n,r)=>n+r.errors,0),referenceUnits=rows.reduce((n,r)=>n+r.referenceUnits,0);rates[language]={errors,referenceUnits,rate:errors/referenceUnits}}
 summary.arms[name]={stage:run.stage??'real-time-coordinator-stop-to-review',rates,latencyMs:distribution(perCase.map(r=>r.elapsedMs)),nonSpeechEmissions:perCase.filter(r=>r.referenceUnits===0&&r.text.trim()).length,...(run.workerPeakRssBytes?{workerPeakRssBytes:run.workerPeakRssBytes}:{}),perCase};
}
summary.limitations=['Development-only evaluation; no new held-out-test claim.','Pipeline memory values in the original driver report cover the Go process; packaged-offline worker RSS is reported separately.','This is single-backend validation. Comparison with historical ONNX latency is descriptive, not a new interleaved paired treatment.','Actual Windows, physical Intel and clean minimum-OS acceptance remain open.'];
fs.writeFileSync(output,JSON.stringify(summary,null,2)+'\n',{flag:'wx'});
for(const [name,arm] of Object.entries(summary.arms)) console.log(name,JSON.stringify({...arm,perCase:undefined}));
