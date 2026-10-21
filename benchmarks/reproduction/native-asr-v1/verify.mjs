// Reuse the reviewed benchmark archive reader for checksum and path verification.
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { openArchive } from '../../../tools/evaluation/archive/manifest.mjs';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const archive=await openArchive(path.join(root,'benchmarks/results/native-asr-v1/archive.json'),root);
if(archive.manifest.status!=='complete'||archive.manifest.entries.length===0)throw Error('Incomplete native ASR archive');
console.log(await archive.verify());
