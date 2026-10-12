import { spawnSync } from 'node:child_process';
import { readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const mode = process.argv[2];
if (!['core', 'desktop', 'format'].includes(mode)) throw new Error('Use core, desktop, or format');
const modules = mode === 'format' ? ['packages/core', 'apps/desktop'] : [mode === 'core' ? 'packages/core' : 'apps/desktop'];
function sourceFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    if (['build', 'dist'].includes(entry.name)) return [];
    const name = path.join(directory, entry.name);
    return entry.isDirectory() ? sourceFiles(name) : entry.name.endsWith('.go') ? [name] : [];
  });
}
function run(command, args, cwd, env, capture = false) {
  const result = spawnSync(command, args, { cwd, env, encoding: 'utf8', stdio: capture ? 'pipe' : 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    if (capture) console.error(result.stderr);
    process.exit(result.status ?? 1);
  }
  return result.stdout ?? '';
}
for (const module of modules) {
  const cwd = path.join(root, module);
  const env = { ...process.env, ...(module === 'packages/core' ? { GOWORK: 'off' } : {}) };
  const files = sourceFiles(cwd);
  if (mode === 'format') run('gofmt', ['-w', ...files], cwd, env);
  else {
    const unformatted = run('gofmt', ['-l', ...files], cwd, env, true).trim();
    if (unformatted) { console.error(`Unformatted Go files:\n${unformatted}`); process.exit(1); }
    run('go', ['test', './...'], cwd, env);
    run('go', ['vet', './...'], cwd, env);
  }
}
