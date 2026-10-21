import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {errorCounts,distribution,NORMALIZATION} from '../../../tools/evaluation/src/metrics.mjs';
const [corpusFile,root,out]=process.argv.slice(2);
if(!out)throw Error('usage: report.mjs corpus root new-output');
const read=p=>JSON.parse(fs.readFileSync(p,'utf8'));
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const lines=p=>fs.readFileSync(p,'utf8').trim().split('\n').map(JSON.parse);
const corpus=read(corpusFile),samples=corpus.samples.filter(s=>s.split==='dev'),summary={normalization:NORMALIZATION,corpusSha256:sha(corpusFile),split:'dev',cases:samples.length,arms:{}};
for(const name of ['mlx17-8bit','mlx17-4bit','mlx06-4bit']){
 const dir=path.join(root,name),run=read(path.join(dir,'run.json'));
 if(run.status!=='complete'||run.corpusSha256!==summary.corpusSha256)throw Error('Incomplete/mismatched run');
 for(const [file,key] of [['results.jsonl','resultsSha256'],['warmups.jsonl','warmupsSha256']])if(sha(path.join(dir,file))!==run[key])throw Error('Result hash mismatch');
 const rows=lines(path.join(dir,'results.jsonl'));
 if(rows.length!==samples.length||new Set(rows.map(x=>x.caseId)).size!==samples.length)throw Error('Incomplete or duplicate observations');
 const perCase=samples.map(s=>{const r=rows.find(x=>x.caseId===s.id);if(!r||r.audioSha256!==s.audio.sha256)throw Error('Missing or changed audio');return {...r,language:s.language,...errorCounts(s.reference,r.text,s.unit)};});
 const rates={};for(const lang of ['zh','en','mixed']){const group=perCase.filter(s=>s.language===lang&&s.referenceUnits>0),errors=group.reduce((n,s)=>n+s.errors,0),units=group.reduce((n,s)=>n+s.referenceUnits,0);rates[lang]={errors,referenceUnits:units,rate:errors/units};}
 summary.arms[name]={stage:run.stage,revision:run.revision,rates,latencyMs:distribution(perCase.map(x=>x.elapsedMs)),tokenLimitHits:perCase.filter(x=>x.tokenLimitReached).length,nonSpeechEmissions:perCase.filter(x=>x.referenceUnits===0&&x.text.trim()).length,computeRtf:perCase.reduce((n,x)=>n+x.elapsedMs,0)/samples.reduce((n,x)=>n+x.audio.durationMs,0),weightBytes:run.weightBytes,perCase};
}
const nativeDir=path.join(root,'native-paired'),nativeRun=read(path.join(nativeDir,'run.json'));
if(nativeRun.status!=='complete'||nativeRun.corpusHash!==crypto.createHash('sha256').update(JSON.stringify(corpus)).digest('hex'))throw Error('Incomplete native run');
for(const [file,key]of [['results.jsonl','resultsSha256'],['warmups.jsonl','warmupsSha256']])if(sha(path.join(nativeDir,file))!==nativeRun[key])throw Error('Native hash mismatch');
const nativeRows=lines(path.join(nativeDir,'results.jsonl'));
for(const variant of nativeRun.config.variants){
 const rows=nativeRows.filter(x=>x.variantId===variant.id);
 if(rows.length!==samples.length||new Set(rows.map(x=>x.caseId)).size!==samples.length)throw Error('Unpaired native rows');
 const perCase=samples.map(s=>{const r=rows.find(x=>x.caseId===s.id);if(!r)throw Error('Missing native case');return {caseId:s.id,text:r.response.finalText,language:s.language,elapsedMs:r.metrics.stopToReviewMs,...errorCounts(s.reference,r.response.finalText,s.unit)};});
 const rates={};for(const lang of ['zh','en','mixed']){const group=perCase.filter(s=>s.language===lang&&s.referenceUnits>0),errors=group.reduce((n,s)=>n+s.errors,0),units=group.reduce((n,s)=>n+s.referenceUnits,0);rates[lang]={errors,referenceUnits:units,rate:errors/units};}
 summary.arms[variant.id]={stage:'native-stop-to-review',rates,latencyMs:distribution(perCase.map(x=>x.elapsedMs)),nonSpeechEmissions:perCase.filter(x=>x.referenceUnits===0&&x.text.trim()).length,perCase};
}
summary.paired={};
for(const [candidate,baseline]of [['mlx17-4bit','mlx17-8bit'],['mlx06-4bit','mlx17-4bit'],['xasr-160-qwen3asr-final','xasr-160-qwen3asr-17-final']]){
 const rows=summary.arms[candidate].perCase.map(r=>{const b=summary.arms[baseline].perCase.find(x=>x.caseId===r.caseId);return {caseId:r.caseId,deltaErrors:r.errors-b.errors,sameText:r.text===b.text};});
 summary.paired[candidate]={baseline,improved:rows.filter(x=>x.deltaErrors<0).length,worsened:rows.filter(x=>x.deltaErrors>0).length,sameErrorCount:rows.filter(x=>x.deltaErrors===0).length,sameText:rows.filter(x=>x.sameText).length};
}
fs.writeFileSync(out,JSON.stringify(summary,null,2)+'\n',{flag:'wx'});
for(const [name,arm]of Object.entries(summary.arms))console.log(name,JSON.stringify({...arm,perCase:undefined}));
