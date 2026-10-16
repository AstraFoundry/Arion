import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const commands = new Map([
  ['desktop', { command: process.execPath, args: [path.join(root, 'scripts/desktop-command.mjs'), 'build'] }],
  ['site', { command: 'pnpm', args: ['site:build'] }],
]);
const [component, ...extra] = process.argv.slice(2);
if (extra.length || !commands.has(component)) throw new Error('Usage: node scripts/build.mjs <desktop|site>');
const { command, args } = commands.get(component);
// Windows launches the pinned pnpm through its command shim; all arguments are fixed.
const result = spawnSync(command, args, { cwd: root, stdio: 'inherit', shell: command === 'pnpm' && process.platform === 'win32' });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
