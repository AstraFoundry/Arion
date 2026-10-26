import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { readFileSync, createWriteStream, createReadStream } from 'node:fs';
import { createHash } from 'node:crypto';
import path from 'node:path';
const [worker, model, output] = process.argv.slice(2);
const modelDigest = createHash('sha256');
for await (const chunk of createReadStream(model)) modelDigest.update(chunk);
const modelSha256 = modelDigest.digest('hex');
if (modelSha256 !== '5f219b78a1d9c3b9e97da27708b36f8a0bc1bfc1650b541c0a6dbaf87c9a62d0') throw new Error('Model hash mismatch');
const corpus = JSON.parse(readFileSync('benchmarks/corpora/asr-v1/corpus.json'));
const selected = ['aishell-BAC009S0766W0137', 'libri-251-137823-0017'].map(id => corpus.samples.find(sample => sample.id === id));
const audio = selected.map(sample => {
  const file = readFileSync(path.join('benchmarks/corpora/asr-v1', sample.audio.path));
  if (createHash('sha256').update(file).digest('hex') !== sample.audio.sha256) throw new Error('Fixture hash mismatch');
  for (let offset = 12; offset + 8 <= file.length;) {
    const name = file.toString('ascii', offset, offset + 4), size = file.readUInt32LE(offset + 4);
    if (name === 'data') return file.subarray(offset + 8, offset + 8 + size);
    offset += 8 + size + (size & 1);
  }
  throw new Error('Missing WAV data');
});
const log = createWriteStream(output, { flags: 'wx' });
const record = value => { const line = JSON.stringify(value); log.write(line + '\n'); console.log(line); };
const child = spawn(worker, ['worker', model, '--threads', '4', '--no-warmup'], { stdio: ['pipe', 'pipe', 'ignore'], windowsHide: true, env: { ...process.env, SPEECH_NO_UPDATE_CHECK: '1' } });
const pending = [], queued = [];
const lines = createInterface({ input: child.stdout });
lines.on('line', line => { const message = JSON.parse(line); if (pending.length) pending.shift()(message); else queued.push(message); });
let exitCode;
const exited = new Promise(resolve => child.on('exit', code => { exitCode = code; resolve(code); }));
const next = async () => {
  let timer;
  try { return await Promise.race([queued.length ? Promise.resolve(queued.shift()) : new Promise(resolve => pending.push(resolve)), new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Worker reply timeout')), 60000); }), exited.then(code => { throw new Error('Worker exited ' + code); })]); }
  finally { clearTimeout(timer); }
};
const send = value => child.stdin.write(JSON.stringify(value) + '\n');
const terminal = async id => { for (;;) { const message = await next(); if (message.id === id && ['end', 'cancelled', 'error'].includes(message.type)) return message; } };
record({ event: 'provenance', driverSha256: createHash('sha256').update(readFileSync(new URL(import.meta.url))).digest('hex'), workerSha256: createHash('sha256').update(readFileSync(worker)).digest('hex'), modelSha256, audioSha256: selected.map(sample => sample.audio.sha256), threads: 4, cancellationDeadlineMs: 100, adapterTimeoutApplied: false });
try {
  const ready = await next();
  if (ready.type !== 'ready') throw new Error('Not ready');
  record({ event: 'ready', protocol: ready.protocol });
  for (let cycle = 0; cycle < 2; cycle++) {
    const id = 'cancel-' + cycle;
    const start = performance.now();
    send({ type: 'chunk', id, seq: 0, pcm: Buffer.concat(Array(5).fill(audio[0])).toString('base64') });
    send({ type: 'transcribe', id, sample_rate: 16000 });
    const timeout = setTimeout(() => { record({ event: 'cancel-request', cycle, elapsedMs: performance.now() - start }); send({ type: 'cancel', id }); }, 100);
    const answer = await terminal(id);
    clearTimeout(timeout);
    record({ event: 'cancel', cycle, elapsedMs: performance.now() - start, answer });
    const reuse = 'reuse-' + cycle, reusedAt = performance.now();
    send({ type: 'chunk', id: reuse, seq: 0, pcm: audio[1].toString('base64') });
    send({ type: 'transcribe', id: reuse, sample_rate: 16000 });
    const reused = await terminal(reuse);
    record({ event: 'reuse', cycle, elapsedMs: performance.now() - reusedAt, answer: reused });
  }
  child.stdin.end();
  record({ event: 'exit', code: await exited });
} finally {
  if (exitCode === undefined) child.kill();
  log.end();
}
