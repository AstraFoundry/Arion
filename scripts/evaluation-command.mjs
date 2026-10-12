import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const command = process.argv[2];
const names = { plan: ['CONFIG', 'PLAN'], validate: ['CORPUS'], run: ['CONFIG', 'CORPUS', 'DRIVER', 'OUT'], report: ['RESULTS'], e2e: ['SUITE', 'DRIVER', 'OUT'] };
if (!names[command]) throw new Error('Unknown evaluation command');
const args = [path.join(root, 'tools/evaluation/cli.mjs'), command];
for (const name of names[command]) {
  const value = process.env[name];
  if (!value) { console.error(`${name} is required for make ${command}; see tools/evaluation/README.md`); process.exit(1); }
  args.push(`--${name === 'PLAN' ? 'out' : name.toLowerCase()}`, value);
}
const result = spawnSync(process.execPath, args, { cwd: root, stdio: 'inherit', shell: false });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
