import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {patchCorrectionIdentity,verifiedCorrectionIdentity} from './keyboard-correction-patch.mjs';
const source=readFileSync('benchmarks/corpora/ime-spelling-reference-v1/upstream-syllabifier.cc','utf8');
test('exact upstream patch is idempotent and changes only three lines',()=>{
 const patched=patchCorrectionIdentity(source);
 assert.equal(patchCorrectionIdentity(patched),patched);
 const before=source.split('\n'),after=patched.split('\n');assert.equal(before.length,after.length);
 assert.equal(before.filter((line,i)=>line!==after[i]).length,3);
});
test('reject partial patch, duplicate sites and source drift',()=>{
 assert.throws(()=>patchCorrectionIdentity(source.replace('set<SyllableId> exact_match_syllables;','set<pair<SyllableId,size_t>> exact_match_syllables;')));
 assert.throws(()=>patchCorrectionIdentity(source+'\nexact_match_syllables.insert(m.value);'));
 assert.throws(()=>patchCorrectionIdentity(source.replace('exact_match_syllables.find(m.value)','exact_match_syllables.count(m.value)')));
});

const pin=JSON.parse(readFileSync('scripts/keyboard-runtime.json','utf8')).patches.correctionIdentity;
test('full-source pins reject unrelated drift and incorrect patch identities',()=>{
 const patched=verifiedCorrectionIdentity(source,pin);
 assert.equal(verifiedCorrectionIdentity(patched,pin),patched);
 assert.throws(()=>verifiedCorrectionIdentity(source+'\n',pin),/source identity/);
 assert.throws(()=>verifiedCorrectionIdentity(source,{...pin,version:2}),/source identity/);
 assert.throws(()=>verifiedCorrectionIdentity(source,{...pin,patchedSha256:'0'.repeat(64)}),/patch output/);
});
