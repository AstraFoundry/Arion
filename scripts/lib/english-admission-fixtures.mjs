import {createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import path from 'node:path';
export function englishAdmissionFixtures(root) {
  const directory=path.join(root,'benchmarks/corpora/ime-english-admission-v1');
  const provenance=JSON.parse(readFileSync(path.join(directory,'provenance.json'),'utf8'));
  return provenance.fixtures.map(fixture=>{
    if (!['development.tsv','validation.tsv','holdout.tsv'].includes(fixture.file)) throw new Error('Unknown English admission fixture.');
    const file=path.join(directory,fixture.file);
    if(createHash('sha256').update(readFileSync(file)).digest('hex')!==fixture.sha256) throw new Error('English admission fixture changed.');
    return file;
  });
}
