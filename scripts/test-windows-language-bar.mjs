// Native COM contract checks only; no TSF registration or input-profile changes.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'win32') throw new Error('Language bar contracts require native Windows.');
if (process.argv.length !== 2) throw new Error('Usage: node scripts/test-windows-language-bar.mjs');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const output = path.join(root, '.workspace/tests/windows-language-bar');
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'run-'));
const reportPath = path.join(directory, 'report.json');
const digest = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const compiler = process.env.CXX ?? 'g++';
const files = ['language_bar.h', 'language_bar_test.cpp', 'identity.h', 'lifetime.h', 'native_copy.h'].map(name => `apps/desktop/ime/windows/${name}`);
files.push('scripts/test-windows-language-bar.mjs');
const report = { schema: 1, boundary: 'Native isolated language-bar COM contracts with mock ITfMenu/ITfLangBarItemSink and real GDI icons; no registration, activation, popup display or system input-menu acceptance claim', directory, sources: {}, commands: [], status: 'running' };
const save = () => writeFileSync(reportPath, `${JSON.stringify(report, null, 2)}\n`);
function run(phase, command, args) {
  const result = spawnSync(command, args, { cwd: directory, encoding: 'utf8', windowsHide: true, timeout: 120000, maxBuffer: 8 * 1024 * 1024 });
  report.commands.push({ phase, command, args, status: result.status, error: result.error?.message, stdout: result.stdout, stderr: result.stderr });
  save();
  process.stdout.write(result.stdout ?? ''); process.stderr.write(result.stderr ?? '');
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${phase} failed (${result.status}).`);
  return result;
}
try {
  for (const file of files) report.sources[file] = digest(path.join(root, file));
  report.compiler = run('compiler-version', compiler, ['--version']).stdout.split('\n')[0].trim();
  const binary = path.join(directory, 'language-bar-test.exe');
  run('compile', compiler, ['-std=c++17', '-DINITGUID', '-O2', '-Wall', '-Wextra', '-Werror', '-static', path.join(root, 'apps/desktop/ime/windows/language_bar_test.cpp'), '-o', binary, '-lole32', '-loleaut32', '-luuid', '-ladvapi32', '-lgdi32']);
  report.binarySha256 = digest(binary);
  run('native-contract', binary, []);
  for (const [file, expected] of Object.entries(report.sources)) if (digest(path.join(root, file)) !== expected) throw new Error(`Source changed during check: ${file}`);
  report.status = 'passed';
} catch (error) { report.status = 'failed'; report.error = error.message; process.exitCode = 1; }
finally { save(); console.log(`Windows language bar evidence: ${reportPath}`); }
if (report.status !== 'passed') throw new Error(report.error);
