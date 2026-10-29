import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { Driver } from '../src/protocol.mjs';

async function fixture(t, source, timeoutMs = 2000) {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-driver-transport-'));
  const filename = path.join(directory, 'process.mjs');
  await writeFile(filename, source);
  const driver = new Driver({ executable: process.execPath, args: [filename], timeoutMs }, directory);
  t.after(async () => {
    await driver.close();
    await rm(directory, { recursive: true, force: true });
  });
  return driver;
}

const echo = `import { createInterface } from 'node:readline';
createInterface({input:process.stdin}).on('line',line=>{
  const request=JSON.parse(line);
  console.log(JSON.stringify({protocol:1,requestId:request.requestId,execution:'fixture',op:request.op}));
});`;

test('persistent transport owns increasing IDs and refuses concurrent requests', async t => {
  const driver = await fixture(t, echo);
  const first = driver.request({ op: 'hello', requestId: 'caller-id', protocol: 99 });
  assert.throws(() => driver.request({ op: 'concurrent' }), /sequential/);
  assert.deepEqual(await first, { protocol: 1, requestId: '1', execution: 'fixture', op: 'hello' });
  assert.equal((await driver.request({ op: 'again' })).requestId, '2');
});

test('invalid JSON and stale request IDs permanently reject the transport', async t => {
  for (const body of ["console.log('not JSON')", "console.log(JSON.stringify({protocol:1,requestId:'old'}))"]) {
    const driver = await fixture(t, `import {createInterface} from 'node:readline';
createInterface({input:process.stdin}).on('line',()=>{${body};});`);
    await assert.rejects(driver.request({ op: 'hello' }), /JSONL/);
    await assert.rejects(driver.request({ op: 'again' }), /JSONL/);
  }
});

test('unsolicited stdout cannot be consumed as a later request response', async t => {
  const driver = await fixture(t, `console.log(JSON.stringify({protocol:1,requestId:'1'})); setInterval(()=>{},1000);`);
  await new Promise(resolve => driver.lines.once('line', resolve));
  await assert.rejects(driver.request({ op: 'hello' }), /Unsolicited/);
});

test('driver exit and deadline failure remain terminal', async t => {
  const exiting = await fixture(t, 'process.exit(3);');
  await assert.rejects(exiting.request({ op: 'hello' }), /exited/);
  const silent = await fixture(t, 'setInterval(()=>{},1000);', 50);
  await assert.rejects(silent.request({ op: 'hello' }), /timed out/);
  await assert.rejects(silent.request({ op: 'again' }), /timed out/);
});

test('spawn failures reject requests and still allow confirmed shutdown', async t => {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-driver-missing-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const driver = new Driver({ executable: path.join(directory, 'missing'), args: [], timeoutMs: 2000 }, directory);
  await assert.rejects(driver.request({ op: 'hello' }), /could not start/);
  await driver.close();
  await assert.rejects(driver.request({ op: 'again' }), /could not start/);
});

test('oversized protocol lines are rejected before accepting a response', async t => {
  const driver = await fixture(t, `import {createInterface} from 'node:readline';
createInterface({input:process.stdin}).on('line',()=>process.stdout.write('x'.repeat(8*1024*1024+1)));`);
  await assert.rejects(driver.request({ op: 'hello' }), /exceeds limit/);
});

test('closing a driver settles in-flight work and waits for process shutdown', async t => {
  const driver = await fixture(t, 'setInterval(()=>{},1000);');
  const failed = assert.rejects(driver.request({ op: 'hello' }), /exited/);
  await driver.close();
  await failed;
  assert.ok(driver.process.exitCode !== null || driver.process.signalCode !== null);
});
