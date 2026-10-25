// Read-only Windows client provenance and protocol compatibility validation.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'win32') throw new Error('This gate requires native Windows.');
const args = process.argv.slice(2);
if (args.some(arg => arg !== '--require-existing') || args.length > 1) throw new Error('Usage: node scripts/test-windows-ime-client.mjs [--require-existing]');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const output = path.join(root, '.workspace/tests/windows-ime-client');
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'run-'));
const report = { schema: 1, directory, status: 'running', boundary: 'Protocol-2 admission policy; isolated unique pipe real process/token proof; retained protected f9 file/module/closure proof and rejection of its old protocol, not protocol-2 native acceptance. No registration, installation, TSF activation, UAC or user profile writes. DllMain-only mapping; macOS compile only.', sources: {}, commands: [] };
const digest = file => createHash('sha256').update(readFileSync(file)).digest('hex');
for (const folder of ['imebridge', 'tsf']) {
  for (const entry of readdirSync(path.join(root, 'apps/desktop/infrastructure', folder))) {
    if (entry.endsWith('.go')) {
      const relative = `apps/desktop/infrastructure/${folder}/${entry}`;
      report.sources[relative] = digest(path.join(root, relative));
    }
  }
}
report.sources['scripts/test-windows-ime-client.mjs'] = digest(fileURLToPath(import.meta.url));
const save = () => writeFileSync(path.join(directory, 'report.json'), `${JSON.stringify(report, null, 2)}\n`);
function run(phase, argv, extra = {}) {
  const execution = spawnSync(process.env.GO ?? 'go', argv, { cwd: path.join(root, 'apps/desktop'), encoding: 'utf8', windowsHide: true, timeout: 180000, maxBuffer: 16 * 1024 * 1024, env: { ...process.env, ARION_REQUIRE_RETAINED_CLIENT: args.includes('--require-existing') ? '1' : '0', ...extra } });
  report.commands.push({ phase, command: 'go', args: argv, environment: extra, status: execution.status, error: execution.error?.message, stdout: execution.stdout, stderr: execution.stderr });
  save();
  process.stdout.write(execution.stdout ?? '');
  process.stderr.write(execution.stderr ?? '');
  if (execution.error || execution.status !== 0) throw execution.error ?? new Error(`${phase} failed (${execution.status})`);
}
try {
  run('go-version', ['version']);
  run('windows-real-proof-and-races', ['test', '-race', '-json', '-count=1', './infrastructure/imebridge', './infrastructure/tsf']);
  run('darwin-bridge-compile-only', ['test', '-c', './infrastructure/imebridge', '-o', path.join(directory, 'imebridge-darwin.test')], { GOOS: 'darwin', GOARCH: 'arm64', CGO_ENABLED: '0' });
  report.darwinBinarySHA256 = digest(path.join(directory, 'imebridge-darwin.test'));
  report.status = 'passed';
} catch (error) {
  report.status = 'failed'; report.error = error.message; throw error;
} finally { save(); process.stdout.write(`Client verification evidence: ${path.join(directory, 'report.json')}\n`); }
