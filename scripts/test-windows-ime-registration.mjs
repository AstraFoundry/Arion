// Pure registration contracts and read-only queries; no UAC or OS mutation.
// The verify cases use synthetic file bytes and never load a payload DLL.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'win32') throw new Error('Windows IME registration checks require native Windows.');
const args = process.argv.slice(2);
if (args.length && (args.length !== 1 || args[0] !== '--require-existing')) throw new Error('Usage: node scripts/test-windows-ime-registration.mjs [--require-existing]');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'apps/desktop/ime/windows');
const output = path.join(root, '.workspace/tests/windows-ime-registration');
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'run-'));
const reportPath = path.join(directory, 'report.json');
const compiler = process.env.CXX ?? 'g++';
const sha = bytes => createHash('sha256').update(bytes).digest('hex');
const digest = file => sha(readFileSync(file));
const tests = ['registration', 'selection', 'user_install', 'category', 'migration', 'upgrade', 'control'];
const files = ['control.cpp', 'loader.h', 'identity.h', 'wire.h','wire_text.h', 'payload.h', 'keyboard_payload.h', 'json.h', ...tests.map(name => `${name}_test.cpp`)];
const report = { schema: 1, boundary: 'Callback-only profile migration contracts, read-only existing-state query, synthetic file-byte verifier fixtures; no installation, UAC, TSF activation or payload execution', directory, sources: {}, commands: [], status: 'running' };
const save = () => writeFileSync(reportPath, `${JSON.stringify(report, null, 2)}\n`);
function run(phase, command, argv, allowed = [0]) {
  const result = spawnSync(command, argv, { cwd: directory, encoding: 'utf8', windowsHide: true, timeout: 120000, maxBuffer: 8 * 1024 * 1024 });
  report.commands.push({ phase, command, args: argv, status: result.status, error: result.error?.message, stdout: result.stdout, stderr: result.stderr });
  save();
  process.stdout.write(result.stdout ?? '');
  process.stderr.write(result.stderr ?? '');
  if (result.error) throw result.error;
  if (!allowed.includes(result.status)) throw new Error(`${phase} failed (${result.status}).`);
  return result;
}
try {
  for (const name of files) report.sources[`apps/desktop/ime/windows/${name}`] = digest(path.join(source, name));
  report.sources['scripts/test-windows-ime-registration.mjs'] = digest(fileURLToPath(import.meta.url));
  report.compiler = run('compiler-version', compiler, ['--version']).stdout.split('\n')[0].trim();
  const flags = ['-std=c++17', '-DINITGUID', '-O2', '-Wall', '-Wextra', '-Werror', '-static', '-municode'];
  const links = ['-lole32', '-loleaut32', '-luuid', '-ladvapi32', '-lshell32'];
  const stagedLoader=path.join(root,'apps/desktop/resources/windows-amd64/ime/ArionTSFLoader.dll');
  const loaderFlags=existsSync(stagedLoader)?[`-DARION_TSF_LOADER_SHA256="${digest(stagedLoader)}"`]:[];
  for (const name of tests) {
    const binary = path.join(directory, `${name}-test.exe`);
    run(`compile-${name}`, compiler, [...flags, ...loaderFlags, path.join(source, `${name}_test.cpp`), '-o', binary, ...links]);
    const execution = run(name === 'control' ? 'read-only-existing-profile-query' : `callback-contract-${name}`, binary, [], name === 'control' && !args.length ? [0, 77] : [0]);
    report[`${name}BinarySha256`] = digest(binary);
    if (name === 'control') report.existingProfileQuery = execution.status === 0 ? 'passed' : 'not-run-no-existing-registration';
  }
  const payload = path.join(directory, 'synthetic-payload');
  const ime = path.join(payload, 'ime');
  const keyboard = path.join(payload, 'keyboard');
  mkdirSync(ime, { recursive: true });
  const sourceDLL = path.join(ime, 'ArionTSF.dll');
  writeFileSync(sourceDLL, 'Synthetic file-hash fixture; never executed.\n');
  const sourceLoader=path.join(ime,'ArionTSFLoader.dll');
  writeFileSync(sourceLoader,'Synthetic stable loader hash fixture; never executed.\n');
  const entries = {};
  for (const member of ['lib/ArionKeyboard.dll', 'lib/ArionRime.dll', 'data/build/arion_pinyin.schema.yaml']) {
    const file = path.join(keyboard, member);
    mkdirSync(path.dirname(file), { recursive: true });
    const bytes = Buffer.from(`Synthetic static verifier fixture: ${member}\n`);
    writeFileSync(file, bytes);
    entries[member] = { sha256: sha(bytes), sizeBytes: bytes.length };
  }
  const manifestPath = path.join(keyboard, 'runtime.json');
  const manifest = JSON.stringify({ fixture: 'Synthetic bytes; no executable validity or product readiness claim', files: entries });
  writeFileSync(manifestPath, manifest);
  const boundHash = digest(manifestPath);
  const helper = path.join(directory, 'arion-ime-readonly-test.exe');
  run('compile-bound-read-only-helper', compiler, [...flags, `-DARION_KEYBOARD_MANIFEST_SHA256="${boundHash}"`, `-DARION_TSF_LOADER_SHA256="${digest(sourceLoader)}"`, path.join(source, 'control.cpp'), '-o', helper, ...links]);
  report.verifyHelperSha256 = digest(helper);
  report.boundSyntheticManifestSha256 = boundHash;
  const inventory = current => Object.fromEntries(readdirSync(current, { withFileTypes: true }).flatMap(entry => {
    const file = path.join(current, entry.name);
    return entry.isDirectory() ? Object.entries(inventory(file)) : [[path.relative(payload, file).replaceAll('\\', '/'), digest(file)]];
  }));
  const before = inventory(payload);
  run('verify-valid-synthetic-closure', helper, ['verify', sourceDLL, digest(sourceDLL)]);
  if (JSON.stringify(inventory(payload)) !== JSON.stringify(before)) throw new Error('Read-only verifier changed its valid fixture.');
  run('verify-refuses-incorrect-source-hash', helper, ['verify', sourceDLL, '0'.repeat(64)], [1]);
  if (JSON.stringify(inventory(payload)) !== JSON.stringify(before)) throw new Error('Rejected source verification changed its fixture.');
  writeFileSync(manifestPath, `${manifest}\n`);
  const tampered = inventory(payload);
  run('verify-refuses-manifest-different-from-compiled-binding', helper, ['verify', sourceDLL, digest(sourceDLL)], [1]);
  if (JSON.stringify(inventory(payload)) !== JSON.stringify(tampered)) throw new Error('Rejected manifest verification changed its fixture.');
  writeFileSync(manifestPath, manifest);
  writeFileSync(sourceLoader,'Changed synthetic loader; never executed.\n');
  run('verify-refuses-unbound-loader',helper,['verify',sourceDLL,digest(sourceDLL)],[1]);
  writeFileSync(sourceLoader,'Synthetic stable loader hash fixture; never executed.\n');
  report.syntheticFixtureIdentities = before;
  for (const [name, expected] of Object.entries(report.sources)) if (digest(path.join(root, name)) !== expected) throw new Error(`Source changed during registration checks: ${name}`);
  report.status = 'passed';
} catch (error) { report.status = 'failed'; report.error = error.message; process.exitCode = 1; }
finally { save(); console.log(`Windows IME registration evidence: ${reportPath}`); }
if (report.status !== 'passed') throw new Error(report.error);
