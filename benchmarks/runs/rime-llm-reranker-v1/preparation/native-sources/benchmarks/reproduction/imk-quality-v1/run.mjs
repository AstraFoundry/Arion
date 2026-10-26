// Real engine benchmark. Preparation and timed execution are explicit separate steps.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync, appendFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { performance } from 'node:perf_hooks';
import { buildIMELearning } from '../../../scripts/lib/build-ime-learning.mjs';
import { evaluate } from './evaluate.mjs';

if (process.platform !== 'darwin' || process.arch !== 'arm64') throw new Error('This baseline requires native arm64 macOS; other hosts need a separately recorded run.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const mode = process.argv[2], build = process.argv[3] && path.resolve(process.argv[3]), out = process.argv[4] && path.resolve(process.argv[4]);
if (!['prepare', 'run'].includes(mode) || !build || (mode === 'run' ? !out || process.argv.length !== 5 : process.argv.length !== 4)) throw new Error('usage: run.mjs prepare BUILD_DIRECTORY | run BUILD_DIRECTORY NEW_OUTPUT_DIRECTORY');
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const json = file => JSON.parse(readFileSync(file, 'utf8'));
const save = (file, value) => writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
const corpusFile = path.join(root, 'benchmarks/corpora/imk-quality-v1/corpus.json');
const corpus = json(corpusFile);
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const binary = path.join(build, 'driver');
const required = ['benchmarks/reproduction/imk-quality-v1/driver.mm', 'benchmarks/reproduction/imk-quality-v1/run.mjs',
  'benchmarks/reproduction/imk-quality-v1/evaluate.mjs', 'benchmarks/corpora/imk-quality-v1/corpus.json', 'benchmarks/corpora/imk-quality-v1/provenance.json',
  'apps/desktop/ime/macos/keyboard.m', 'apps/desktop/ime/macos/keyboard.h', 'apps/desktop/ime/macos/learning.mm',
  'apps/desktop/ime/macos/learning.h', 'apps/desktop/ime/macos/learning_crypto.swift', 'scripts/lib/build-ime-learning.mjs',
  'scripts/keyboard-runtime.json', 'apps/desktop/resources/darwin/keyboard/runtime.json'];
const command = (name, args) => execFileSync(name, args, { cwd: root, encoding: 'utf8', maxBuffer: 8 * 1024 * 1024 }).trim();
const probe = (name, args) => {
  const result = spawnSync(name, args, { encoding: 'utf8' });
  return { status: result.status, stdout: result.stdout, stderr: result.stderr, error: result.error?.message ?? null };
};
const metadata = () => ({
  corpusSha256: hash(corpusFile), runtimeManifestSha256: hash(path.join(runtime, 'runtime.json')),
  nativeLibrarySha256: hash(path.join(runtime, 'lib/librime.1.dylib')), nativeExecutableSha256: hash(binary),
  architecture: { host: 'arm64', runner: process.arch, executable: command('lipo', ['-archs', binary]), translation: 'none' },
  hardware: { model: command('sysctl', ['-n', 'hw.model']), cpu: command('sysctl', ['-n', 'machdep.cpu.brand_string']),
    cpus: command('sysctl', ['-n', 'hw.ncpu']), memoryBytes: command('sysctl', ['-n', 'hw.memsize']) },
  operatingSystem: command('sw_vers', []), xcode: command('xcrun', ['clang', '--version']), swift: command('xcrun', ['swiftc', '--version']),
  powerSettings: probe('pmset', ['-g', 'custom']), powerSource: probe('pmset', ['-g', 'batt']), thermal: probe('pmset', ['-g', 'therm']),
  gitRevision: command('git', ['rev-parse', 'HEAD']), gitDirtyState: command('git', ['status', '--porcelain']),
  timer: 'mach_absolute_time, mach_timebase_info conversion to milliseconds', compileFlags: ['arm64-apple-macos14.0', '-O2', '-fobjc-arc', '-std=c++17 for driver/backend', '-DARION_LEARNING_TESTING'],
  engine: 'librime1.17.0 + production ArionKeyboardEngine + arion_pinyin + encrypted synthetic-key backend',
  coldDefinition: 'Fresh process/engine/session and empty encrypted profile; no OS-cache eviction. Native initialization timing excludes process/dynamic-loader time.',
  warmDefinition: 'Fresh process/profile as cold, then raw xuetang/heishan warmups cancelled before measurement. Engine resident; no committed warmup or test-derived learned memory.',
  controls: { networkInference: false, keychain: false, installedIME: false, foregroundUI: false, warmupInputs: ['xuetang', 'heishan'], seeds: [1701, 1702, 1703], hostLoadAverage: os.loadavg(), isolatedCPU: false },
});
if (mode === 'prepare') {
  if (existsSync(build)) throw new Error('Build directory exists; preserve previous attempts.');
  mkdirSync(build, { recursive: true });
  save(path.join(build, 'status.json'), { status: 'preparing' });
  const sources = {};
  for (const name of required) {
    const file = path.join(root, name), destination = path.join(build, 'sources', name);
    mkdirSync(path.dirname(destination), { recursive: true }); copyFileSync(file, destination); sources[name] = hash(file);
  }
  save(path.join(build, 'source-identities.json'), sources);
  const log = path.join(build, 'compile.log');
  const compile = args => {
    const result = spawnSync('xcrun', args, { cwd: root, encoding: 'utf8' });
    appendFileSync(log, `${JSON.stringify(['xcrun', ...args])}\n${result.stdout ?? ''}${result.stderr ?? ''}`);
    if (result.error || result.status !== 0) throw new Error(`Native compilation failed; original log: ${log}`);
  };
  try {
    const learning = buildIMELearning({ root, outputDir: build, arch: 'arm64', testing: true });
    const target = learning.target, native = path.join(root, 'apps/desktop/ime/macos'), objects = [];
    for (const [file, objcxx] of [[path.join(native, 'keyboard.m'), false], [path.join(root, required[0]), true]]) {
      const object = path.join(build, `native-${objects.length}.o`);
      compile([objcxx ? 'clang++' : 'clang', ...(objcxx ? ['-std=c++17'] : []), '-target', target, '-O2', '-fobjc-arc',
        '-Wall', '-Wextra', '-Werror', '-DARION_LEARNING_TESTING', '-DGLOG_USE_GLOG_EXPORT', '-I', native,
        '-I', path.join(runtime, 'include'), '-isystem', path.join(root, '.workspace/keyboard-runtime/sdk/include'), '-c', file, '-o', object]);
      objects.push(object);
    }
    compile([learning.linker, '-target', target, ...objects, ...learning.objects, path.join(runtime, 'lib/librime.1.dylib'),
      '-Xlinker', '-rpath', '-Xlinker', path.join(runtime, 'lib'), '-framework', 'Cocoa', '-framework', 'InputMethodKit', ...learning.linkArgs, '-o', binary]);
    const frozen = metadata(); frozen.sourceHashes = sources;
    save(path.join(build, 'metadata.json'), frozen);
    save(path.join(build, 'status.json'), { status: 'prepared', nativeExecutableSha256: frozen.nativeExecutableSha256 });
    console.log(`Prepared actual native driver and froze100 fixtures; no timed runs: ${build}`);
  } catch (error) {
    save(path.join(build, 'status.json'), { status: 'prepare-failed', error: error.message }); throw error;
  }
} else {
  if (existsSync(out)) throw new Error('Output directory exists; never overwrite a benchmark attempt.');
  const frozen = json(path.join(build, 'metadata.json'));
  if (json(path.join(build, 'status.json')).status !== 'prepared' || hash(binary) !== frozen.nativeExecutableSha256) throw new Error('Prepared native executable changed.');
  for (const [name, expected] of Object.entries(frozen.sourceHashes)) if (hash(path.join(root, name)) !== expected) throw new Error(`Frozen source changed: ${name}`);
  if (hash(path.join(runtime, 'lib/librime.1.dylib')) !== frozen.nativeLibrarySha256) throw new Error('Pinned native runtime changed.');
  mkdirSync(out, { recursive: true });
  for (const name of ['requests', 'raw', 'profiles']) mkdirSync(path.join(out, name));
  save(path.join(out, 'status.json'), { status: 'running' });
  const current = metadata(); current.preparationMetadata = frozen;
  save(path.join(out, 'metadata.json'), current);
  const index = [], plan = [];
  for (const seed of [1701, 1702, 1703]) {
    let state = seed;
    const random = () => { state ^= state << 13; state ^= state >>> 17; state ^= state << 5; return (state >>> 0) / 4294967296; };
    const cases = [...corpus.cases];
    for (let i = cases.length - 1; i > 0; i--) { const j = Math.floor(random() * (i + 1)); [cases[i], cases[j]] = [cases[j], cases[i]]; }
    for (const fixture of cases) for (const stratum of random() < .5 ? ['cold-process', 'warm-process'] : ['warm-process', 'cold-process']) plan.push({ seed, id: fixture.id, stratum });
  }
  save(path.join(out, 'plan.json'), plan);
  try {
    for (const row of plan) {
      const fixture = corpus.cases.find(fixture => fixture.id === row.id), name = `${row.seed}-${row.id}-${row.stratum}`;
      const request = `requests/${row.id}-${row.stratum}.json`, result = `raw/${name}.json`;
      const requestFile = path.join(out, request), resultFile = path.join(out, result);
      if (!existsSync(requestFile)) save(requestFile, { id: fixture.id, input: fixture.input, warmups: row.stratum === 'cold-process' ? [] : ['xuetang', 'heishan'] });
      const profile = path.join(out, 'profiles', name);
      const start = performance.now();
      const child = spawnSync(binary, [path.join(runtime, 'data'), profile, requestFile, resultFile], { cwd: root, encoding: 'utf8', timeout: 30000 });
      const childWallMs = performance.now() - start;
      appendFileSync(path.join(out, 'driver.log'), `${JSON.stringify({ ...row, exitStatus: child.status, signal: child.signal, error: child.error?.message ?? null })}\n${child.stdout ?? ''}${child.stderr ?? ''}`);
      if (!existsSync(resultFile)) throw new Error(`Native result missing: ${name}`);
      index.push({ ...row, result, request, sha256: hash(resultFile), exitStatus: child.status, childWallMs });
      save(path.join(out, 'observations.json'), index);
      if (child.error || child.status !== 0 || json(resultFile).status !== 'completed') throw new Error(`Native process failed: ${name}`);
      rmSync(profile, { recursive: true, force: true });
      if (index.length % 100 === 0) console.log(`Completed ${index.length}/600 actual native processes.`);
    }
    for (const [name, expected] of Object.entries(frozen.sourceHashes)) if (hash(path.join(root, name)) !== expected) throw new Error(`Source changed during timed execution: ${name}`);
    save(path.join(out, 'status.json'), { status: 'completed', completedProcesses: index.length, finalLoadAverage: os.loadavg() });
    save(path.join(out, 'report.json'), evaluate(out, corpusFile));
    console.log(`Completed600 real runs and independent scoring: ${out}`);
  } catch (error) {
    save(path.join(out, 'status.json'), { status: 'failed', completedProcesses: index.length, error: error.message }); throw error;
  }
}
