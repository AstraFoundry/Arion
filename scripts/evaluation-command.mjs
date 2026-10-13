import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const command = process.argv[2];
const names = { plan: ['CONFIG', 'PLAN'], validate: ['CORPUS'], run: ['CONFIG', 'CORPUS', 'DRIVER', 'OUT'], report: ['RESULTS'], e2e: ['SUITE', 'DRIVER', 'OUT'] };
const rewrite = command?.startsWith('rewrite-');
const operation = rewrite ? command.slice('rewrite-'.length) : command;
const rewriteNames = { plan: ['REWRITE_CONFIG', 'REWRITE_PLAN'], validate: ['CORPUS'], run: ['REWRITE_CONFIG', 'CORPUS', 'DRIVER', 'OUT'], report: ['RESULTS'], 'human-export': ['RESULTS', 'OUT'], 'human-report': ['RESULTS', 'REVIEW', 'RATINGS'] };
const required = (rewrite ? rewriteNames : names)[operation];
if (!required) throw new Error('Unknown evaluation command');
const args = [path.join(root, rewrite ? 'tools/evaluation/rewrite-cli.mjs' : 'tools/evaluation/cli.mjs'), operation];
const optionNames = { PLAN: 'out', REWRITE_PLAN: 'out', REWRITE_CONFIG: 'config' };
for (const name of required) {
  const value = process.env[name];
  if (!value) { console.error(`${name} is required for make ${command}; see tools/evaluation/README.md`); process.exit(1); }
  args.push(`--${optionNames[name] ?? name.toLowerCase()}`, value);
}
if (rewrite && operation === 'human-export') for (const [name, option] of [['RATERS', 'raters'], ['RATING_SEED', 'seed']]) if (process.env[name]) args.push(`--${option}`, process.env[name]);
const result = spawnSync(process.execPath, args, { cwd: root, stdio: 'inherit', shell: false });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
