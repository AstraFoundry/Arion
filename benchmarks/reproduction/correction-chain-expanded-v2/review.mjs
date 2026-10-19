// Export every delivered output alongside evaluator-only intent after inference.
import fs from 'node:fs';
import path from 'node:path';
import { observations } from './observations.mjs';
const [directory,out]=process.argv.slice(2);
if (!out) throw new Error('Usage: review.mjs RUN NEW_TEXT');
const data=observations(path.resolve(directory));
const groups=[...new Set(data.rows.map(r=>r.caseId))].map(caseId=>{
 const label=data.labels[caseId];
 return [caseId,`SPEECH ${label.transcript}`,`INTENT ${label.references[0]}`,
  ...data.rows.filter(r=>r.caseId===caseId).map(r=>`${r.arm}: ${JSON.stringify(r.text)}${r.truncated?' [TRUNCATED]':''}${r.unavailable?' [UNAVAILABLE]':''}`)].join('\n');
});
fs.writeFileSync(out,groups.join('\n\n')+'\n',{flag:'wx'});
