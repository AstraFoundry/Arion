import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, readFile, access, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { runScenarios, assertObservations } from '../src/e2e.mjs';
import { hashFile } from '../src/artifacts.mjs';

const cli = fileURLToPath(new URL('../cli.mjs', import.meta.url));
const suite = { version: 1, scenarios: [{ id: 'owned-editor', acceptance: 'EVAL-A02',
  capabilities: ['synthetic-editor'], actions: [{ op: 'type', text: 'owned text' }],
  assertions: { 'editor.text': 'owned text', commitCount: 1 } }] };

async function fixture(t, mode = 'ok') {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-e2e-contract-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const artifact = path.join(directory, 'artifact.txt');
  await writeFile(artifact, 'synthetic transport artifact; no native execution');
  const source = `import {createInterface} from 'node:readline';
import {appendFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
const mode=${JSON.stringify(mode)};
createInterface({input:process.stdin}).on('line',line=>{
 const r=JSON.parse(line); appendFileSync('requests.jsonl',JSON.stringify(r)+'\\n');
 const out={protocol:1,requestId:r.requestId,execution:mode==='fixture'?'fixture':'live'};
 if(r.op==='hello')Object.assign(out,{artifacts:r.artifacts,platform:'darwin',capabilities:mode==='capability'?[]:['synthetic-editor']});
 if(r.op==='scenario'){
  const trace='synthetic owned trace';writeFileSync('trace.txt',trace);
  Object.assign(out,{fixtureOwnership:'isolated',observations:{editor:{text:mode==='assertion'?'wrong':'owned text'},commitCount:1},
   evidence:mode==='absent'?[]:[{path:'trace.txt',sha256:mode==='tamper'?'0'.repeat(64):createHash('sha256').update(trace).digest('hex')}]});
 }
 if(r.op==='cleanup'){writeFileSync('cleanup.txt','requested');out.cleanupComplete=mode!=='cleanup';}
 console.log(JSON.stringify(out));
});`;
  const script = path.join(directory, 'process.mjs');
  await writeFile(script, source);
  const config = { executable: process.execPath, args: [script], timeoutMs: 2000,
    artifacts: [{ id: 'runtime', path: 'artifact.txt', sha256: await hashFile(artifact), license: 'synthetic-test' }] };
  return { directory, config, output: path.join(directory, 'output') };
}

test('scenario contract verifies evidence, keeps assertions local and records cleanup', async t => {
  const f = await fixture(t);
  assert.equal(await runScenarios(suite, f.config, f.directory, f.output), 1);
  const result = JSON.parse(await readFile(path.join(f.output, 'results.json'), 'utf8'));
  assert.equal(result.status, 'passed');
  assert.equal(result.results[0].response.observations.commitCount, 1);
  assert.equal(result.cleanup.cleanupComplete, true);
  const requests = (await readFile(path.join(f.directory, 'requests.jsonl'), 'utf8')).trim().split('\n').map(JSON.parse);
  assert.equal(requests.length, 3);
  assert.deepEqual(requests[1].actions, suite.scenarios[0].actions);
  assert.equal(Object.hasOwn(requests[1], 'assertions'), false);
  assert.equal(Object.hasOwn(requests[1], 'acceptance'), false);
  await assert.rejects(runScenarios(suite, f.config, f.directory, f.output), /EEXIST/);
});

test('failed provenance, capability, evidence and assertions retain failure and cleanup', async t => {
  for (const [mode, error] of [['fixture', /provenance/], ['capability', /capability/],
    ['absent', /evidence missing/], ['tamper', /checksum/], ['assertion', /assertion failed/], ['cleanup', /cleanup/]]) {
    const f = await fixture(t, mode);
    await assert.rejects(runScenarios(suite, f.config, f.directory, f.output), error);
    const result = JSON.parse(await readFile(path.join(f.output, 'results.json'), 'utf8'));
    assert.equal(result.status, 'failed', mode);
    await access(path.join(f.directory, 'cleanup.txt'));
  }
});

test('scenario assertions and suite identity cannot pass with missing observations', async t => {
  assert.throws(() => assertObservations(suite.scenarios[0], { status: 'passed' }), /assertion/);
  const f = await fixture(t);
  await assert.rejects(runScenarios({ ...suite, scenarios: [...suite.scenarios, suite.scenarios[0]] }, f.config, f.directory, f.output), /Duplicate/);
  await assert.rejects(access(f.output), /ENOENT/);
});

test('public CLI rejects research commands and incomplete or duplicate native inputs', async () => {
  for (const args of [['plan'], ['e2e'], ['e2e', '--suite', 'one', '--suite', 'two'], ['e2e', '--corpus', 'one']]) {
    const result = spawnSync(process.execPath, [cli, ...args], { encoding: 'utf8' });
    assert.equal(result.status, 1);
    assert.match(result.stderr, /Product scenarios failed/);
    assert.equal(result.stdout, '');
  }
});
