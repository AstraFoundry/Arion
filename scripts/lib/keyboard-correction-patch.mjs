import {createHash} from 'node:crypto';
// Exact matches must identify both the spelling and its consumed raw length.
export function patchCorrectionIdentity(source) {
 const replacements=[
  ['set<SyllableId> exact_match_syllables;','set<pair<SyllableId,size_t>> exact_match_syllables;'],
  ['exact_match_syllables.insert(m.value);','exact_match_syllables.insert({m.value,m.length});'],
  ['exact_match_syllables.find(m.value)','exact_match_syllables.find({m.value,m.length})'],
 ];
 const original=replacements.every(([from,to])=>source.split(from).length===2&&!source.includes(to));
 const patched=replacements.every(([from,to])=>source.split(to).length===2&&!source.includes(from));
 if(patched)return source;
 if(!original)throw new Error('Pinned Rime correction identity patch does not match the complete expected source.');
 return replacements.reduce((text,[from,to])=>text.replace(from,to),source);
}

export function verifiedCorrectionIdentity(source, pin) {
 const digest=text=>createHash('sha256').update(text).digest('hex');
 if(pin?.version!==1 || ![pin.originalSha256,pin.patchedSha256].includes(digest(source)))
  throw new Error('Pinned Rime correction source identity differs.');
 const patched=patchCorrectionIdentity(source);
 if(digest(patched)!==pin.patchedSha256)throw new Error('Pinned Rime correction patch output differs.');
 return patched;
}
