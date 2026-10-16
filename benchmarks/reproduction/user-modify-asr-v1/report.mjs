// Recompute recognition metrics from complete, hashed model observations.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {errorCounts, distribution, normalize} from '../../../tools/evaluation/src/metrics.mjs';
const [fixtures, dir, output] = process.argv.slice(2);
if (!fixtures || !dir || !output) throw new Error('usage: report.mjs fixtures run output');
const read = p => JSON.parse(fs.readFileSync(p,'utf8'));
const sha = p => crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const run=read(path.join(dir,'run.json')), corpus=read(path.join(fixtures,'corpus.json'));
if(run.status!=='complete') throw new Error('Incomplete run');
for(const s of corpus.samples) if(sha(path.join(fixtures,s.audio.path))!==s.audio.sha256) throw new Error('Changed audio '+s.id);
if(sha(path.join(fixtures,'warmup.wav'))!==run.warmupAudioSha256) throw new Error('Changed warmup audio');
for(const [file,digest] of Object.entries(run.inputHashes)) if(sha(path.join(fixtures,file))!==digest) throw new Error('Changed input '+file);
if(sha(path.join(dir,'results.jsonl'))!==run.resultsSha256 || sha(path.join(dir,'warmups.jsonl'))!==run.warmupsSha256) throw new Error('Changed observations');
const rows=fs.readFileSync(path.join(dir,'results.jsonl'),'utf8').trim().split('\n').map(JSON.parse);
const by=new Map();
for(const r of rows){const key=r.caseId+'/'+r.arm;if(by.has(key))throw new Error('Duplicate observation');by.set(key,r);}
if(rows.length!==corpus.samples.length*run.experiment.arms.length)throw new Error('Unpaired coverage');
const includes=(text,term)=>{
 if(!term)return null;
 const t=normalize(text),word=normalize(term);
 return /[\p{Script=Han}]/u.test(word)?t.includes(word):new RegExp(`(?<![a-z0-9])${word.replace(/[.*+?^${}()|[\]\\]/g,'\\$&')}(?![a-z0-9])`,'u').test(t);
};
const perCase=[];
for(const s of corpus.samples){
 const base=by.get(s.id+'/none');if(!base)throw new Error('Missing baseline');
 for(const arm of run.experiment.arms){
  const r=by.get(s.id+'/'+arm);if(!r)throw new Error('Missing arm');
  const score=errorCounts(s.reference,r.text,s.unit),b=errorCounts(s.reference,base.text,s.unit);
  perCase.push({caseId:s.id,kind:s.kind,language:s.language,arm,text:r.text,reference:s.reference,term:s.term,termHit:includes(r.text,s.term),...score,baselineErrors:b.errors,deltaErrors:score.errors-b.errors,elapsedMs:r.elapsedMs,tokenLimitReached:r.tokenLimitReached,promptTokens:r.promptTokens});
 }
}
const summary={status:'complete',model:run.experiment.model,sourceStage:run.stage,cases:corpus.samples.length,observations:rows.length,selectedIds:run.selectedIds,normalization:'arion-nfkc-lower-punctuation-space-v1',semanticReview:'not_independently_reviewed',conversionParity:'unmeasured',arms:{},perCase};
for(const arm of run.experiment.arms){
 const all=perCase.filter(x=>x.arm===arm),speech=all.filter(x=>x.referenceUnits>0),noise=all.filter(x=>x.kind==='nonspeech');
 const rates={};for(const lang of ['zh','en','mixed']){const group=speech.filter(x=>x.language===lang),errors=group.reduce((n,x)=>n+x.errors,0),units=group.reduce((n,x)=>n+x.referenceUnits,0);rates[lang]={cases:group.length,errors,referenceUnits:units,rate:units?errors/units:null};}
 const groups={};for(const kind of new Set(all.map(x=>x.kind))){const g=all.filter(x=>x.kind===kind),terms=g.filter(x=>x.term);groups[kind]={cases:g.length,termHits:terms.filter(x=>x.termHit).length,termCases:terms.length,improved:g.filter(x=>x.deltaErrors<0).length,worsened:g.filter(x=>x.deltaErrors>0).length};}
 summary.arms[arm]={rates,groups,improved:speech.filter(x=>x.deltaErrors<0).length,worsened:speech.filter(x=>x.deltaErrors>0).length,baselineCorrectCorrupted:speech.filter(x=>x.baselineErrors===0&&x.errors>0).length,baselineCorrectCases:speech.filter(x=>x.baselineErrors===0).length,nonSpeechEmissions:noise.filter(x=>x.text.trim()).length,nonSpeechCases:noise.length,truncations:all.filter(x=>x.tokenLimitReached).length,latencyMs:distribution(all.map(x=>x.elapsedMs)),promptTokens:distribution(all.map(x=>x.promptTokens)),peakRssBytes:Math.max(...rows.filter(x=>x.arm===arm).map(x=>x.peakRssBytes)),computeRtf:all.reduce((n,x)=>n+x.elapsedMs,0)/corpus.samples.reduce((n,x)=>n+x.audio.durationMs,0)};
}
fs.writeFileSync(output,JSON.stringify(summary,null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(summary.arms,null,2));
