// Entity-aware scoring of verified, complete acoustic observations.
import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';
import {errorCounts,normalize,distribution} from '../../../tools/evaluation/src/metrics.mjs';
const [fixtures,dir,out]=process.argv.slice(2);
const read=p=>JSON.parse(fs.readFileSync(p,'utf8'));
const sha=p=>crypto.createHash('sha256').update(fs.readFileSync(p)).digest('hex');
const run=read(path.join(dir,'run.json')),corpus=read(path.join(fixtures,'corpus.json')),memory=read(path.join(fixtures,'memory.json'));
if(run.status!=='complete')throw new Error('Incomplete run');
for(const [file,digest] of Object.entries(run.inputHashes))if(sha(path.join(fixtures,file))!==digest)throw new Error('Input changed');
for(const [file,key] of [['results.jsonl','resultsSha256'],['calls.jsonl','callsSha256'],['warmups.jsonl','warmupsSha256']])if(sha(path.join(dir,file))!==run[key])throw new Error('Observation changed');
const lines=name=>fs.readFileSync(path.join(dir,name),'utf8').trim().split('\n').map(JSON.parse);
const calls=lines('calls.jsonl'),rows=lines('results.jsonl'),byCall=new Map(calls.map(c=>[c.callId,c])),by=new Map();
if(byCall.size!==calls.length||calls.length!==run.modelCalls)throw new Error('Invalid call provenance');
for(const r of rows){const key=r.caseId+'/'+r.arm;if(by.has(key))throw new Error('Duplicate row');by.set(key,r);const ids=r.callIds.map(id=>byCall.get(id));if(ids.some(c=>!c||c.caseId!==r.caseId)||ids.at(-1).text!==r.text)throw new Error('Output does not match actual call');}
if(rows.length!==run.caseIds.length*run.arms.length||rows.length!==run.observationCount)throw new Error('Incomplete pairing');
const escape=s=>s.replace(/[.*+?^${}()|[\]\\]/g,'\\$&');
const present=(text,term)=>{const t=text.normalize('NFKC').toLowerCase(),w=term.normalize('NFKC').toLowerCase();return /[\u3400-\u9fff]/u.test(w)?t.includes(w):new RegExp('(?<![a-z0-9])'+[...w.replace(/\s/gu,'')].map(escape).join('\\s*')+'(?![a-z0-9])','u').test(t);};
const pairs=new Set(memory.entries.filter(e=>e.approved).map(e=>JSON.stringify([e.source,e.target])));
const eligible=memory.entries.filter(e=>e.approved&&run.config.logicalDay-e.updatedDay<run.config.expireDays&&!pairs.has(JSON.stringify([e.target,e.source])));
const unique=[...new Set(eligible.map(e=>e.target))];
const wrong=(text,reference)=>unique.filter(t=>present(text,t)&&!present(reference,t));
const perCase=[];
for(const s of corpus.samples.filter(s=>run.caseIds.includes(s.id))){
 if(sha(path.join(fixtures,s.audio.path))!==s.audio.sha256)throw new Error('Audio changed');
 const b=by.get(s.id+'/none');if(!b)throw new Error('Missing baseline');
 const bs=errorCounts(s.reference,b.text,s.unit),bw=wrong(b.text,s.reference);
 for(const arm of run.arms){const r=by.get(s.id+'/'+arm);if(!r)throw new Error('Missing arm');const score=errorCounts(s.reference,r.text,s.unit),ew=wrong(r.text,s.reference);perCase.push({...r,reference:s.reference,kind:s.kind,language:s.language,term:s.term,entityHit:s.term?present(r.text,s.term):null,...score,baselineErrors:bs.errors,deltaErrors:score.errors-bs.errors,wrongEntities:ew,newWrongEntities:ew.filter(t=>!bw.includes(t)),retrievalTargetHit:s.term?r.selected.some(h=>present(h.target,s.term)):null});}
}
const summary={status:'complete',cases:run.caseIds.length,observations:rows.length,actualModelCalls:calls.length,arms:{},perCase,semanticReview:'not_independently_reviewed',selectionSha256:run.selectionSha256};
for(const arm of run.arms){const all=perCase.filter(r=>r.arm===arm),speech=all.filter(r=>r.referenceUnits>0),primary=speech.filter(r=>['recurring','selection_miss'].includes(r.kind)),controls=speech.filter(r=>!['recurring','selection_miss','expired_control'].includes(r.kind));const rates={};
 for(const lang of ['zh','en','mixed']){const g=speech.filter(r=>r.language===lang),errors=g.reduce((n,x)=>n+x.errors,0),units=g.reduce((n,x)=>n+x.referenceUnits,0);rates[lang]={errors,referenceUnits:units,rate:units?errors/units:null};}
 summary.arms[arm]={primaryEntityHits:primary.filter(r=>r.entityHit).length,primaryEntityCases:primary.length,newWrongEntityCases:all.filter(r=>r.newWrongEntities.length).length,wrongEntityCases:all.filter(r=>r.wrongEntities.length).length,baselineCorrectRegressed:speech.filter(r=>r.baselineErrors===0&&r.errors>0).length,baselineCorrectCases:speech.filter(r=>r.baselineErrors===0).length,controlWorsened:controls.filter(r=>r.deltaErrors>0).length,controlCases:controls.length,improved:speech.filter(r=>r.deltaErrors<0).length,worsened:speech.filter(r=>r.deltaErrors>0).length,nonspeechEmissions:all.filter(r=>r.kind==='nonspeech'&&r.text.trim()).length,truncations:all.filter(r=>r.tokenLimitReached).length,reddecodeCases:all.filter(r=>r.route==='audio_redecode').length,retrievalCoverage:primary.filter(r=>r.retrievalTargetHit).length,rates,pipelineMs:distribution(all.map(r=>r.pipelineMs)),secondPassMs:distribution(all.map(r=>r.secondPassMs).filter(x=>x>0))};}
fs.writeFileSync(out,JSON.stringify(summary,null,2)+'\n',{flag:'wx'});console.log(JSON.stringify(summary.arms,null,2));
