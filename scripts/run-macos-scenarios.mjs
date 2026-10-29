// Runs the owned macOS product scenarios against the packaged Arion.app.
// Builds the editor fixture and E2E driver, synthesizes audio fixtures once,
// pins every artifact by SHA-256 and invokes the evaluation harness.
// Requires: macOS, a built apps/desktop/build/darwin-<arch>/Arion.app with
// Accessibility access granted, and a model store holding the selected bundle.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync, copyFileSync, rmSync } from 'node:fs';
import { homedir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') {
  console.error('macOS product scenarios require macOS.');
  process.exit(2);
}
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const work = path.join(root, '.workspace', 'e2e');
const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
const app = path.join(root, 'apps', 'desktop', 'build', `darwin-${arch}`, 'Arion.app');
const models = process.env.ARION_E2E_MODELS ?? path.join(homedir(), 'Library', 'Application Support', 'Arion', 'models');
const run = (command, args, options = {}) => execFileSync(command, args, { stdio: 'inherit', ...options });

if (!existsSync(app)) { console.error(`Missing ${app}; run make build desktop first.`); process.exit(1); }
if (!existsSync(models)) { console.error(`Missing model store ${models}; finish onboarding or set ARION_E2E_MODELS.`); process.exit(1); }

// Editor fixture app.
const fixture = path.join(work, 'fixtures', 'ArionEditorFixture.app');
const source = path.join(root, 'apps', 'desktop', 'e2e', 'fixtures', 'editor');
rmSync(fixture, { recursive: true, force: true });
mkdirSync(path.join(fixture, 'Contents', 'MacOS'), { recursive: true });
copyFileSync(path.join(source, 'Info.plist'), path.join(fixture, 'Contents', 'Info.plist'));
run('clang', ['-fobjc-arc', '-O1', '-framework', 'Cocoa', path.join(source, 'main.m'), '-o', path.join(fixture, 'Contents', 'MacOS', 'ArionEditorFixture')]);
run('codesign', ['--force', '--sign', '-', fixture]);

// Driver binary.
const driver = path.join(work, 'bin', 'arion-e2e-driver');
mkdirSync(path.dirname(driver), { recursive: true });
run('go', ['build', '-o', driver, './cmd/arion-e2e-driver'], { cwd: path.join(root, 'apps', 'desktop'), env: { ...process.env, CGO_ENABLED: '1' } });

// Audio fixtures: fixed system voices, converted bit-exactly to 16 kHz mono.
const audio = path.join(work, 'fixtures', 'audio');
mkdirSync(audio, { recursive: true });
const voices = { 'hello-world': ['Samantha', 'Hello world.'], mixed: ['Tingting', '请把这个 PR merge 到 main 分支。'] };
for (const [name, [voice, text]] of Object.entries(voices)) {
  const wav = path.join(audio, `${name}.wav`);
  if (existsSync(wav)) continue;
  const aiff = path.join(audio, `${name}.aiff`);
  run('say', ['-v', voice, '-o', aiff, text]);
  run('ffmpeg', ['-nostdin', '-loglevel', 'error', '-y', '-i', aiff, '-ac', '1', '-ar', '16000', '-sample_fmt', 's16', '-bitexact', wav]);
  rmSync(aiff);
}
const silence = path.join(audio, 'silence.wav');
if (!existsSync(silence)) run('ffmpeg', ['-nostdin', '-loglevel', 'error', '-y', '-f', 'lavfi', '-i', 'anullsrc=r=16000:cl=mono', '-t', '2.5', '-sample_fmt', 's16', '-bitexact', silence]);

const sha = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const rel = file => path.relative(work, file);
const artifacts = [
  ['arion-app', path.join(app, 'Contents', 'MacOS', 'Arion'), 'Arion development build'],
  ['editor-fixture', path.join(fixture, 'Contents', 'MacOS', 'ArionEditorFixture'), 'Arion test fixture'],
  ['e2e-driver', driver, 'Arion test driver'],
  ...['hello-world', 'mixed', 'silence'].map(name => [`audio-${name}`, path.join(audio, `${name}.wav`), 'Synthetic fixture (macOS system voice)']),
].map(([id, file, license]) => ({ id, path: rel(file), sha256: sha(file), license }));
const config = {
  executable: rel(driver), args: ['-config', 'driver.json'], timeoutMs: 600000, artifacts,
  app: rel(app), fixtureApp: rel(fixture), models, evidence: 'evidence',
  audio: { 'hello-world': rel(path.join(audio, 'hello-world.wav')), mixed: rel(path.join(audio, 'mixed.wav')), silence: rel(silence) },
};
writeFileSync(path.join(work, 'driver.json'), JSON.stringify(config, null, 2));
const out = path.join(work, `run-${new Date().toISOString().replace(/[:.]/g, '-')}`);
const result = spawnSync(process.execPath, [path.join(root, 'tools', 'driver', 'cli.mjs'), 'e2e',
  '--suite', path.join(root, 'tests', 'e2e', 'macos-shortcut-scenarios.json'), '--driver', path.join(work, 'driver.json'), '--out', out], { stdio: 'inherit' });
console.log(`Results: ${path.relative(root, out)}/results.json`);
process.exitCode = result.status ?? 1;
