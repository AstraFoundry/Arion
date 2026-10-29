import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
if (process.argv[2] !== 'e2e') throw new Error('Public evaluation supports e2e only. Research commands belong to Arion-evaluation.');
const args = [path.join(root, 'tools/driver/cli.mjs'), 'e2e'];
for (const name of ['SUITE', 'DRIVER', 'OUT']) {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required for make e2e; see tools/driver/README.md`);
  args.push(`--${name.toLowerCase()}`, value);
}
const result = spawnSync(process.execPath, args, { cwd: root, stdio: 'inherit', shell: false });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
