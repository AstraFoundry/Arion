// Pin completed research evidence and shared replay dependencies; never include weights.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hash } from '../correction-chain-v1/report.mjs';
import { verify } from '../correction-chain-v1/archive.mjs';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'../../..');
const destination=path.join(root,'benchmarks/runs/correction-chain-expanded-v2');
function files(directory) {
 return fs.readdirSync(directory,{withFileTypes:true}).flatMap(entry=>{
  if(entry.isSymbolicLink()) throw new Error('Archive does not follow symlinks');
  const file=path.join(directory,entry.name);
  return entry.isDirectory() ? (entry.name==='__pycache__' ? [] : files(file)) : entry.isFile() ? [file] : [];
 });
}
const manifest=path.join(destination,'manifest.json');
if(fs.existsSync(manifest)) throw new Error('Archive manifest already exists');
const selected=[...['benchmarks/corpora/correction-chain-expanded-v2','benchmarks/results/correction-chain-expanded-v2','benchmarks/reproduction/correction-chain-expanded-v2','benchmarks/runs/correction-chain-expanded-v2','benchmarks/reproduction/correction-chain-v1'].flatMap(name=>files(path.join(root,name))),
 ...['benchmarks/reproduction/correction-chain-semantic-v1/rubric.json','benchmarks/reproduction/correction-chain-semantic-v1/report.mjs','tools/evaluation/src/metrics.mjs'].map(name=>path.join(root,name))];
const entries=[...new Set(selected)].sort().map(file=>({path:path.relative(destination,file),bytes:fs.statSync(file).size,sha256:hash(file)}));
fs.writeFileSync(manifest,JSON.stringify({version:1,kind:'correction-chain-expanded-archive-v2',files:entries},null,2)+'\n',{flag:'wx'});
console.log(JSON.stringify(verify(manifest)));
