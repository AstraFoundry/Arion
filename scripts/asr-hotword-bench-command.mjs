// Portable entry points for archived production hotword evidence.
import path from 'node:path';
import {spawnSync} from 'node:child_process';
const root=path.resolve(import.meta.dirname,'..');
const [operation,...args]=process.argv.slice(2);
const base=path.join(root,'benchmarks/reproduction/asr-hotwords-v1');
let script,parameters;
if(operation==='verify'){
 if(args.length)throw Error('verify accepts no arguments');
 script='verify.mjs';parameters=[];
}else if(operation==='report'){
 if(args.length!==2||args[0]!=='--out')throw Error('report requires --out <new-json-file>');
 script='report.mjs';parameters=[path.join(root,'benchmarks/runs/asr-hotwords-v1/paired-production'),path.resolve(args[1]),path.join(root,'benchmarks/runs/asr-hotwords-v1/sensitive-isolation')];
}else throw Error('expected report or verify');
const result=spawnSync(process.execPath,[path.join(base,script),...parameters],{cwd:root,stdio:'inherit'});
if(result.error)throw result.error;
process.exit(result.status??1);
