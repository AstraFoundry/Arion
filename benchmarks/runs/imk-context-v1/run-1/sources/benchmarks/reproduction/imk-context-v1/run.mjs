// Context ranking benchmark: builds the native driver from current production
// sources, replays seeded streams with context ranking off and on, and writes
// raw results, source hashes and the evaluated report to a new directory.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildIMELearning } from '../../../scripts/lib/build-ime-learning.mjs';
import { evaluate } from './evaluate.mjs';

if (process.platform !== 'darwin' || process.arch !== 'arm64') throw new Error('Requires native arm64 macOS.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const out = process.argv[2] && path.resolve(process.argv[2]);
if (!out || process.argv.length !== 3) throw new Error('usage: run.mjs NEW_OUTPUT_DIRECTORY');
if (existsSync(out)) throw new Error('Output directory already exists; results are never overwritten.');
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const save = (file, value) => writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
const corpusFile = path.join(root, 'benchmarks/corpora/imk-context-v1/corpus.json');
const corpus = JSON.parse(readFileSync(corpusFile, 'utf8'));
const source = path.join(root, 'apps/desktop/ime/macos');
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const data = path.join(runtime, 'data');
const sources = ['keyboard.h', 'keyboard.m', 'ranking.h', 'ranking.m', 'typing.h', 'typing.m', 'candidate_panel.h',
  'candidate_panel.m', 'learning.h', 'learning.mm', 'learning_crypto.swift'].map(name => path.join(source, name));
const harness = ['benchmarks/reproduction/imk-context-v1/driver.mm', 'benchmarks/reproduction/imk-context-v1/run.mjs',
  'benchmarks/reproduction/imk-context-v1/evaluate.mjs'].map(name => path.join(root, name));

// Mulberry32: a small seeded generator so streams are reproducible.
function random(seed) {
  return () => {
    seed |= 0; seed = seed + 0x6D2B79F5 | 0;
    let t = Math.imul(seed ^ seed >>> 15, 1 | seed);
    t = t + Math.imul(t ^ t >>> 7, 61 | t) ^ t;
    return ((t ^ t >>> 14) >>> 0) / 4294967296;
  };
}
// Every pair appears `repetitions` times in shuffled order; each appearance is
// its context word followed by its target word.
function stream(seed) {
  const next = random(seed);
  const visits = corpus.pairs.flatMap(pair => Array.from({ length: corpus.repetitions }, () => pair));
  for (let i = visits.length - 1; i > 0; i--) {
    const j = Math.floor(next() * (i + 1));
    [visits[i], visits[j]] = [visits[j], visits[i]];
  }
  const seen = new Map();
  const steps = [];
  for (const pair of visits) {
    const occurrence = (seen.get(pair.id) ?? 0) + 1;
    seen.set(pair.id, occurrence);
    for (const kind of ['context', 'target']) {
      steps.push({ index: steps.length, kind, pair: pair.id, occurrence, pinyin: pair[kind].pinyin, word: pair[kind].word });
    }
  }
  return steps;
}

mkdirSync(out, { recursive: true });
const build = mkdtempSync(path.join(os.tmpdir(), 'arion-context-bench-'));
try {
  const learning = buildIMELearning({ root, outputDir: build, arch: 'arm64', testing: true });
  const objects = [...['keyboard.m', 'ranking.m', 'typing.m', 'candidate_panel.m'].map(name => path.join(source, name)),
    path.join(root, 'benchmarks/reproduction/imk-context-v1/driver.mm')].map((file, i) => {
    const object = path.join(build, `object-${i}.o`);
    const cxx = file.endsWith('.mm');
    execFileSync('xcrun', [cxx ? 'clang++' : 'clang', ...(cxx ? ['-x', 'objective-c++', '-std=c++17'] : []),
      '-target', learning.target, '-fobjc-arc', '-O2', '-DARION_LEARNING_TESTING', '-DGLOG_USE_GLOG_EXPORT',
      '-I', source, '-I', path.join(runtime, 'include'), '-isystem', path.join(root, '.workspace/keyboard-runtime/sdk/include'),
      '-c', file, '-o', object], { cwd: root, stdio: 'inherit' });
    return object;
  });
  const driver = path.join(build, 'driver');
  execFileSync('xcrun', [learning.linker, '-target', learning.target, ...objects, ...learning.objects,
    path.join(runtime, 'lib/librime.1.dylib'), '-Xlinker', '-rpath', '-Xlinker', path.join(runtime, 'lib'),
    '-framework', 'Cocoa', '-framework', 'InputMethodKit', ...learning.linkArgs, '-o', driver], { cwd: root, stdio: 'inherit' });

  const runs = [];
  for (const seed of corpus.seeds) {
    const steps = stream(seed);
    for (const contextRanking of [false, true]) {
      const name = `seed-${seed}-${contextRanking ? 'ranking' : 'baseline'}`;
      const profile = mkdtempSync(path.join(build, 'profile-'));
      const request = path.join(out, `${name}.request.json`);
      const output = path.join(out, `${name}.result.json`);
      save(request, { contextRanking, pageSize: corpus.pageSize, steps });
      execFileSync(driver, [data, profile, request, output], { stdio: 'inherit' });
      runs.push({ seed, contextRanking, request: path.basename(request), result: path.basename(output) });
    }
  }
  const environment = {
    corpusSha256: hash(corpusFile),
    sources: Object.fromEntries([...sources, ...harness].map(file => [path.relative(root, file), hash(file)])),
    runtimeManifestSha256: hash(path.join(runtime, 'runtime.json')),
    librimeSha256: hash(path.join(runtime, 'lib/librime.1.dylib')),
    driverSha256: hash(driver),
    hardware: { model: execFileSync('sysctl', ['-n', 'hw.model'], { encoding: 'utf8' }).trim(),
      cpu: execFileSync('sysctl', ['-n', 'machdep.cpu.brand_string'], { encoding: 'utf8' }).trim() },
    operatingSystem: execFileSync('sw_vers', [], { encoding: 'utf8' }).trim(),
    gitHead: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim(),
    gitDirty: execFileSync('git', ['status', '--porcelain'], { cwd: root, encoding: 'utf8' }).trim().length > 0,
  };
  save(path.join(out, 'run.json'), { benchmark: corpus.id, runs, environment });
  for (const file of [corpusFile, ...harness]) cpSync(file, path.join(out, 'sources', path.relative(root, file)));
  for (const file of sources) cpSync(file, path.join(out, 'sources', path.relative(root, file)));
  const report = evaluate(out);
  save(path.join(out, 'report.json'), report);
  console.log(JSON.stringify(report.summary, null, 2));
} finally {
  rmSync(build, { recursive: true, force: true });
}
