import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {scoreText} from '../../../tools/evaluation/live/score-text.mjs';
import {scoreAudio} from '../../../tools/evaluation/live/score-audio.mjs';
const dir='.workspace/research/doubao-alignment';
const read=async p=>JSON.parse(await fs.readFile(p,'utf8'));
const rows=async p=>(await fs.readFile(p,'utf8')).trim().split('\n').map(JSON.parse);
const sha=async p=>crypto.createHash('sha256').update(await fs.readFile(p)).digest('hex');
if((await read(`${dir}/product-state.json`)).status!=='complete')throw Error('Incomplete suite cannot be published');
const manifest=await read(`${dir}/product-manifest.json`);
if((await read(`${dir}/production-audio-state.json`)).status!=='complete')throw Error('Incomplete production audio');
const production=await read(`${dir}/production-audio-manifest.json`);
for(const [p,h]of Object.entries(production.sources))if(await sha(p)!==h)throw Error(`Changed production source ${p}`);
for(const [n,h]of Object.entries(production.binaries))if(await sha(`${dir}/${n}`)!==h)throw Error(`Changed production binary ${n}`);
const matched=await read(`${dir}/matched-manifest.json`);
for(const [name,hash]of Object.entries(manifest.binaries))if(await sha(`${dir}/${name}`)!==hash)throw Error(`Changed matched binary ${name}`);
const initial=await read(`${dir}/frozen.json`);
const promptPath='apps/desktop/infrastructure/llm/prompt/live.json';
if(await sha(promptPath)!==initial[promptPath])throw Error('Prompt changed');
for(const group of ['artifacts','sources','scorers','asrFiles'])for(const [p,hash] of Object.entries(manifest[group]))if(await sha(p)!==(production.sources[p]??hash))throw Error(`Changed ${group}: ${p}`);
for(const [name,hash]of Object.entries(manifest.binaries))if(await sha(`${dir}/${name}`)!==hash)throw Error(`Changed binary ${name}`);
const sets=[
 ['dev','.workspace/research/multilingual-disfluency/corpus.json','matched-dev-baseline0.jsonl','product-dev-product.jsonl','dev'],
 ['alignment','.workspace/research/rewrite-alignment/corpus.json','matched-alignment-baseline0.jsonl','product-alignment-product.jsonl','regression'],
 ['regression','.workspace/live-heldout-corpus.json','matched-regression-baseline0.jsonl','product-regression-product.jsonl','regression'],
 ['fresh',`${dir}/fresh-test.json`,'fresh-baseline0.jsonl','fresh-product.jsonl','test'],
 ['heldout',`${dir}/heldout.json`,'matched-heldout-baseline0.jsonl','product-heldout-product.jsonl','post-review-regression'],
];
const runs=[],paired=[];
for(const [id,p,b,a,split] of sets){
 const corpus=await read(p),before=scoreText(corpus,await rows(`${dir}/${b}`)),after=scoreText(corpus,await rows(`${dir}/${a}`));
 const lang=new Map(corpus.samples.map(x=>[x.id,x.language]));
 const previous=new Map(before.perCase.map(x=>[x.id,x.pass]));
 const count=list=>({cases:list.length,improved:list.filter(x=>!previous.get(x.id)&&x.pass).length,regressed:list.filter(x=>previous.get(x.id)&&!x.pass).length});
 paired.push({id,...count(after.perCase),byLanguage:Object.fromEntries([...new Set(lang.values())].map(l=>[l,count(after.perCase.filter(x=>lang.get(x.id)===l))]))});
 for(const [arm,file]of [['baseline0',b],['product',a]])runs.push({id:`${id}-${arm}`,kind:'text',split,variant:arm,corpus:p,results:`${dir}/${file}`,corpusSha256:await sha(p),resultsSha256:await sha(`${dir}/${file}`)});
}
const audioCorpus=await read(`${dir}/audio-corpus.json`),audioManifest=await read(`${dir}/audio-fixtures/manifest.json`);
for(const arm of ['baseline0','product'])runs.push({id:`audio-${arm}`,kind:'audio',split:'synthetic-audio',variant:arm,corpus:`${dir}/audio-corpus.json`,manifest:`${dir}/audio-fixtures/manifest.json`,results:`${dir}/audio-${arm}.jsonl`,corpusSha256:await sha(`${dir}/audio-corpus.json`),manifestSha256:await sha(`${dir}/audio-fixtures/manifest.json`),resultsSha256:await sha(`${dir}/audio-${arm}.jsonl`)});
const audioBefore=scoreAudio(audioManifest,audioCorpus,await rows(`${dir}/audio-baseline0.jsonl`)),audioAfter=scoreAudio(audioManifest,audioCorpus,await rows(`${dir}/audio-product.jsonl`));
const prev=new Map(audioBefore.perCase.map(x=>[x.id,x]));
paired.push({id:'audio',cases:audioManifest.samples.length,improved:audioAfter.perCase.filter(x=>!prev.get(x.id).pass&&x.pass).length,regressed:audioAfter.perCase.filter(x=>prev.get(x.id).pass&&!x.pass).length,rawSourceChanged:audioAfter.perCase.filter(x=>prev.get(x.id).raw!==x.raw).length});
const productionBefore=scoreAudio(audioManifest,audioCorpus,await rows(`${dir}/audio-production-baseline0.jsonl`)),productionAfter=scoreAudio(audioManifest,audioCorpus,await rows(`${dir}/audio-production-product.jsonl`));
const previousProduction=new Map(productionBefore.perCase.map(x=>[x.id,x]));
paired[paired.length-1].id='audio-diagnostic-fillers-off';
paired.push({id:'audio-production-fillers-on',cases:audioManifest.samples.length,improved:productionAfter.perCase.filter(x=>!previousProduction.get(x.id).pass&&x.pass).length,regressed:productionAfter.perCase.filter(x=>previousProduction.get(x.id).pass&&!x.pass).length,rawSourceChanged:productionAfter.perCase.filter(x=>previousProduction.get(x.id).raw!==x.raw).length});
for(const arm of ['baseline0','product'])runs.push({id:`audio-production-${arm}`,kind:'audio',split:'synthetic-production-audio',variant:arm+'-production',removeFillers:true,corpus:`${dir}/audio-corpus.json`,manifest:`${dir}/audio-fixtures/manifest.json`,results:`${dir}/audio-production-${arm}.jsonl`,corpusSha256:await sha(`${dir}/audio-corpus.json`),manifestSha256:await sha(`${dir}/audio-fixtures/manifest.json`),resultsSha256:await sha(`${dir}/audio-production-${arm}.jsonl`)});
const output='benchmarks/results/disfluency-alignment-v2';
await fs.mkdir(output,{recursive:false});
const config={host:manifest.host,provenance:{revision:manifest.revision,dirty:manifest.dirty,settings:manifest.settings,manifestSha256:await sha(`${dir}/product-manifest.json`),modelAndRuntime:manifest.artifacts,binaries:{...manifest.binaries,...production.binaries},audioPolicies:{diagnosticRemoveFillers:false,productionRemoveFillers:true,textRemoveFillers:true},productionAudioManifestSha256:await sha(`${dir}/production-audio-manifest.json`),productionAudioSources:production.sources,buildInfo:manifest.buildInfo,matchedManifestSha256:await sha(`${dir}/matched-manifest.json`),promptSha256:initial[promptPath],sources:manifest.sources,scorers:Object.fromEntries(await Promise.all(Object.keys(manifest.scorers).map(async p=>[p,await sha(p)]))),uiSourceSha256:{"apps/desktop/bootstrap/app.go":await sha("apps/desktop/bootstrap/app.go"),"apps/desktop/bootstrap/rewrite_test.go":await sha("apps/desktop/bootstrap/rewrite_test.go")},asrCatalogSha256:manifest.asrCatalogSha256},limitations:['Single-author synthetic text and macOS TTS; no independent semantic review or natural speech','Other desktop workloads were active; timing is descriptive, not a causal speed comparison','No rendered native preview, target editor insertion, Windows execution or actual competitor output comparison','Foreign-language text constraints do not establish recognition coverage or multilingual semantic fidelity'],runs};
const configPath='benchmarks/configs/disfluency-alignment-report-v2.json';
await fs.writeFile(configPath,JSON.stringify(config,null,2)+'\n',{flag:'wx'});
const generated=spawnSync(process.execPath,['tools/evaluation/live/report.mjs',configPath,`${output}/summary.json`],{stdio:'inherit'});if(generated.status!==0)throw Error('report failed');
await fs.writeFile(`${output}/paired.json`,JSON.stringify({method:'Matched case exact-constraint pass; improvement is baseline fail to candidate pass, regression the reverse',comparisons:paired},null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify({paired,audioBefore:{pass:audioBefore.finalPass,latency:audioBefore.stopToFinalMs},audioAfter:{pass:audioAfter.finalPass,latency:audioAfter.stopToFinalMs},productionBefore:{pass:productionBefore.finalPass,latency:productionBefore.stopToFinalMs},productionAfter:{pass:productionAfter.finalPass,latency:productionAfter.stopToFinalMs}},null,2));
