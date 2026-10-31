// Actual librime conversion and the production typing state against AppKit's text store.
import {createHash} from 'node:crypto';
import { execFileSync, spawnSync } from 'node:child_process';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync, realpathSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { englishAdmissionFixtures } from './lib/english-admission-fixtures.mjs';
import { buildIMETypingFixture } from './lib/build-ime-typing-fixture.mjs';

if (process.platform !== 'darwin') throw new Error('Keyboard integration tests require macOS.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const digest=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const sourceFiles=execFileSync('git',['ls-files','-co','--exclude-standard','-z','apps/desktop/ime'],{cwd:root,encoding:'utf8'}).split('\0').filter(Boolean);
const sourceIdentities=Object.fromEntries(sourceFiles.map(file=>[file,digest(path.join(root,file))]));
const options = new Map();
for (let i = 2; i < process.argv.length; i += 2) {
  const name = process.argv[i], value = process.argv[i + 1];
  if (!['--data', '--report', '--arch', '--library'].includes(name) || !value || options.has(name)) {
    throw new Error('Usage: node scripts/test-keyboard-input.mjs [--data PATH] [--report PATH] [--arch arm64|x86_64] [--library EXPERIMENTAL_RIME_LIBRARY]');
  }
  if (name === '--arch' && !['arm64', 'x86_64'].includes(value)) throw new Error('Keyboard test architecture must be arm64 or x86_64.');
  options.set(name, name === '--arch' ? value : path.resolve(value));
}
const data = options.get('--data') ?? path.join(root, 'apps/desktop/resources/darwin/keyboard/data');
const report = options.get('--report') ?? path.join(root, '.workspace/tests/keyboard-input.json');
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const runnerArchitectures = { arm64: 'arm64', x64: 'x86_64' };
const runnerArchitecture = runnerArchitectures[process.arch];
if (!runnerArchitecture) throw new Error(`Unsupported keyboard test runner architecture: ${process.arch}`);
const arch = options.get('--arch') ?? runnerArchitecture;
const experimentalLibrary=options.has('--library')?realpathSync(options.get('--library')):null;
const libraryHash=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const experimentalHash=experimentalLibrary?libraryHash(experimentalLibrary):null;
if(experimentalLibrary&&(arch!==runnerArchitecture||realpathSync(path.join(path.dirname(experimentalLibrary),'librime.1.dylib'))!==experimentalLibrary))throw new Error('Experimental runtime requires native architecture and matching loader install name.');
// Apple's documented Rosetta probe treats ENOENT as native execution:
// https://developer.apple.com/documentation/apple-silicon/about-the-rosetta-translation-environment
// sysctl -i suppresses only unknown OIDs; other command failures still throw.
function translationFor(architecture) {
  const value = execFileSync('/usr/bin/arch', [`-${architecture}`, '/usr/sbin/sysctl', '-in', 'sysctl.proc_translated'],
    { encoding: 'utf8' }).trim();
  if (!['', '0', '1'].includes(value)) throw new Error(`Unexpected Rosetta probe result: ${JSON.stringify(value)}`);
  if (architecture === 'arm64' && value === '1') throw new Error('An arm64 executable cannot report Rosetta translation.');
  return { translated: value === '1', oidPresent: value !== '' };
}
const runnerTranslation = translationFor(runnerArchitecture);
const hostArchitecture = runnerArchitecture === 'arm64' || runnerTranslation.translated ? 'arm64' : 'x86_64';
for (const required of [path.join(data, 'build/arion_pinyin.schema.yaml'),
  path.join(runtime, 'include/rime_api.h'), path.join(runtime, 'lib/librime.1.dylib')]) {
  if (!existsSync(required)) throw new Error(`Required pinned keyboard artifact is missing: ${required}`);
}
mkdirSync(path.dirname(report), { recursive: true });
const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-keyboard-test-'));
let status = 2;
try {
  const binary = buildIMETypingFixture({root,outputDir:temporary,arch,fixture:'tests/typing_test.m'});
  const executableArchitecture = execFileSync('xcrun', ['lipo', '-archs', binary], { encoding: 'utf8' }).trim();
  if (executableArchitecture !== arch) throw new Error(`Built keyboard test architecture differs: ${executableArchitecture}`);
  const executableTranslation = translationFor(arch);
  if (executableTranslation.translated !== (hostArchitecture === 'arm64' && arch === 'x86_64')) {
    throw new Error('Executable translation probe disagrees with the detected host and architecture.');
  }
  const execution = {
    hostArchitecture, runnerArchitecture, executableArchitecture,
    launcher: experimentalLibrary?binary:`/usr/bin/arch -${arch}`,
    ...(experimentalLibrary?{experimentalLibrary,experimentalSha256:experimentalHash}:{}),
    translation: executableTranslation.translated ? 'Rosetta 2' : 'none',
    translationProbe: { oid: 'sysctl.proc_translated', ...executableTranslation },
    sourceIdentities,
    gitRevision: execFileSync('git',['rev-parse','HEAD'],{cwd:root,encoding:'utf8'}).trim(),
    dirty: execFileSync('git',['status','--porcelain'],{cwd:root,encoding:'utf8'}).length>0,
    runtimeManifestSha256: digest(path.join(runtime,'runtime.json')),
    physicalIntelHost: hostArchitecture === 'x86_64',
    installedIMEHostDispatch: false,
  };
  process.stdout.write(`Keyboard test execution: ${JSON.stringify(execution)}\n`);
  const nativeReport = path.join(temporary, 'native-report.json');
  const result = spawnSync(experimentalLibrary?binary:'/usr/bin/arch', [...(experimentalLibrary?[]:[`-${arch}`,binary]), data, path.join(temporary, 'user'), nativeReport, ...englishAdmissionFixtures(root)],
    { cwd: root, stdio: 'inherit', env: {...process.env,...(experimentalLibrary?{DYLD_LIBRARY_PATH:path.dirname(experimentalLibrary)}:{})} });
  if (result.error) throw result.error;
  if (result.signal) throw new Error(`Keyboard test terminated by ${result.signal}`);
  status = result.status ?? 2;
  for (const file of readdirSync(temporary).filter(name => /^(emoji-macos-(light|dark)-[0-2]|shift-macos-(pending|committed)|english-macos-(artificial|can|artifi|hel|github|frontend|xin|shi)(?:-[59])?)\.png$/.test(name))) {
    copyFileSync(path.join(temporary, file), path.join(path.dirname(report), file));
  }
  if (existsSync(nativeReport)) {
    const changed=sourceFiles.filter(file=>digest(path.join(root,file))!==sourceIdentities[file]);
    if(changed.length) throw new Error('IME sources changed during native verification: '+changed.join(', '));
    englishAdmissionFixtures(root);
    execution.binarySha256=digest(binary);
    const resultReport = JSON.parse(readFileSync(nativeReport, 'utf8'));
    if(experimentalLibrary)execution.runtimeVerified=resultReport.nativeLibraryPath===experimentalLibrary&&libraryHash(experimentalLibrary)===experimentalHash;
    if(experimentalLibrary&&!execution.runtimeVerified)resultReport.passed=false;
    writeFileSync(report, `${JSON.stringify({ ...resultReport, execution }, null, 2)}\n`);
    if(experimentalLibrary&&!execution.runtimeVerified)throw new Error('Actual experimental runtime identity mismatch.');
  } else if (status === 0) {
    throw new Error('Keyboard test succeeded without producing its result report.');
  }
  process.stdout.write(`Keyboard test report: ${report}\n`);
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
process.exitCode = status;
