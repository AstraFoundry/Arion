// Aggregate explicit semantic review, preserving uncertainty and paired discordance.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { observations, read, root } from './observations.mjs';
import { hash } from '../correction-chain-v1/report.mjs';
import { validateRatings } from '../correction-chain-semantic-v1/report.mjs';
import { distribution, errorCounts } from '../../../tools/evaluation/src/metrics.mjs';
export function summarize(rows) {
  return {cases:rows.length, preserved:rows.filter(r=>r.fidelity==='preserved').length,
    degraded:rows.filter(r=>r.fidelity==='degraded').length, uncertain:rows.filter(r=>r.fidelity==='uncertain').length,
    ready:rows.filter(r=>r.ready).length, strictClean:rows.filter(r=>r.fidelity==='preserved' && r.organization===2).length,
    truncated:rows.filter(r=>r.truncated).length, unavailable:rows.filter(r=>r.unavailable).length,
    serialEngineEstimateMs:distribution(rows.map(r=>r.serialEngineEstimateMs))};
}
export function paired(rows, base, treatment, metric) {
  const selected = rows.filter(r=>r.arm===base && r.task!=='silence');
  const ids = new Map(rows.filter(r=>r.arm===treatment).map(r=>[r.caseId,r]));
  const passes = r=>metric==='ready' ? r.ready : r.fidelity==='preserved';
  const gains=[], losses=[];
  for (const a of selected) {
    const b=ids.get(a.caseId);
    if (!b) throw new Error('Missing paired output');
    if (!passes(a) && passes(b)) gains.push(a.caseId);
    if (passes(a) && !passes(b)) losses.push(a.caseId);
  }
  return {spokenCases:selected.length, gains, losses, net:gains.length-losses.length,
    percentagePointDifference:100*(gains.length-losses.length)/selected.length};
}
export function report(directory, ratingPath) {
  const original=observations(directory), review=read(ratingPath);
  if (original.runSha256!==review.runSha256) throw new Error('Review run mismatch');
  const rubric=path.join(root,'benchmarks/reproduction/correction-chain-semantic-v1/rubric.json');
  if (hash(rubric)!==review.rubricSha256) throw new Error('Review rubric mismatch');
  validateRatings(original,review);
  const identical=new Map();
  for (const r of review.ratings) {
    const key=`${r.caseId}/${r.textSha256}`, grade=`${r.fidelity}/${r.organization}`;
    if (identical.has(key) && identical.get(key)!==grade) throw new Error(`Inconsistent identical-output review: ${r.caseId}`);
    identical.set(key,grade);
  }
  const ratings=new Map(review.ratings.map(r=>[`${r.caseId}/${r.arm}`,r]));
  const rows=original.rows.map(r=>({...r,...ratings.get(`${r.caseId}/${r.arm}`)}));
  const arms={};
  for (const arm of original.run.config.arms) {
    const chosen=rows.filter(r=>r.arm===arm), spoken=chosen.filter(r=>r.task!=='silence');
    arms[arm]={all:summarize(chosen), spoken:summarize(spoken),
      byLanguage:Object.fromEntries(['zh','en','mixed'].map(l=>[l,summarize(spoken.filter(r=>r.language===l))])),
      byVoice:Object.fromEntries([...new Set(spoken.map(r=>r.speakerId))].map(v=>[v,summarize(spoken.filter(r=>r.speakerId===v))])),
      byTask:Object.fromEntries([...new Set(spoken.map(r=>r.task))].map(t=>[t,summarize(spoken.filter(r=>r.task===t))]))};
    if (['asr','asr_correct','offline'].includes(arm)) arms[arm].acousticByLanguage=Object.fromEntries(['zh','en','mixed'].map(language=>{
      const scores=spoken.filter(r=>r.language===language).map(r=>errorCounts(original.labels[r.caseId].transcript,r.text,{zh:'cer',en:'wer',mixed:'mer'}[language]));
      const errors=scores.reduce((n,r)=>n+r.errors,0), referenceUnits=scores.reduce((n,r)=>n+r.referenceUnits,0);
      return [language,{errors,referenceUnits,rate:errors/referenceUnits}];
    }));
  }
  const comparisons={};
  for (const [base,treatment] of [['asr','asr_correct'],['asr','asr_llm'],['asr_llm','asr_correct_llm'],['asr_llm','offline_llm']]) {
    comparisons[`${base} -> ${treatment}`]={...Object.fromEntries(['fidelity','ready'].map(metric=>[metric,paired(rows,base,treatment,metric)])),
      byLanguage:Object.fromEntries(['zh','en','mixed'].map(l=>[l,Object.fromEntries(['fidelity','ready'].map(metric=>[metric,paired(rows.filter(r=>r.language===l),base,treatment,metric)]))]))};
  }
  return {kind:'correction-chain-expanded-semantic-v2',runSha256:original.runSha256,ratingsSha256:hash(ratingPath),
    rubricSha256:review.rubricSha256,scorerSha256:hash(fileURLToPath(import.meta.url)),reviewer:review.reviewer,
    independence:review.independence,arms,paired:comparisons,rows,
    sensitivity:{
      excludeLowRecognitionVoice:{postHoc:true, reason:'Eddy Mandarin voice has low baseline recognition; this is a secondary exclusion, not a changed primary cohort',
        arms:Object.fromEntries(original.run.config.arms.map(arm=>[arm,summarize(rows.filter(r=>r.arm===arm && r.task!=='silence' && r.speakerId!=='Eddy (中文（中国大陆）)'))]))},
      uncertainIsNotPreserved:true, strictCleanIsOrganizationTwo:true,
      technicalSpellingBoundaryCases:['expanded-mixed-30','expanded-mixed-22','expanded-en-21']
    },
    limitations:['96 novel spoken scripts plus four silence controls; original 26 cases excluded',
      'One author and synthetic voices; related themes, not independent spontaneous speakers',
      'Single agent semantic ratings; not independent human adjudication',
      'Timing is sequential engine estimate, not observed native speech-to-insertion',
      'No population confidence or model-family generalization from this diagnostic']};
}
if (process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  const [run,ratings,out]=process.argv.slice(2);
  if (!out) throw new Error('Usage: report.mjs RUN RATINGS NEW_JSON');
  const result=report(path.resolve(run),path.resolve(ratings));
  fs.writeFileSync(out,JSON.stringify(result,null,2)+'\n',{flag:'wx'});
  console.log(JSON.stringify({arms:Object.fromEntries(Object.entries(result.arms).map(([k,v])=>[k,v.spoken])),paired:result.paired},null,2));
}
