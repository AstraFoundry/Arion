// Explicit macOS release return and running-binary diagnostics; never builds or kills.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import { macOSDesktopProcesses, requireDesktopStopped } from './lib/desktop-development.mjs';

if (process.platform !== 'darwin') throw new Error('Desktop session switching currently requires macOS.');
const operation = process.argv[2];
if (!['status', 'production'].includes(operation) || process.argv.length !== 3) throw new Error('Use desktop-session.mjs status|production');
const processes = macOSDesktopProcesses();
for (const p of processes) {
  const sha256 = createHash('sha256').update(readFileSync(p.executable)).digest('hex');
  console.log(JSON.stringify({ ...p, sha256 }));
}
if (!processes.length) console.log('Arion is stopped.');
if (operation === 'production') {
  requireDesktopStopped(processes);
  const bundle = '/Applications/Arion.app';
  if (!existsSync(bundle)) throw new Error('Install the signed release in /Applications/Arion.app first.');
  execFileSync('codesign', ['--verify', '--deep', '--strict', '-R', 'anchor apple generic and identifier "si.astrafoundry.arion"', bundle], { stdio: 'inherit' });
  execFileSync('open', [bundle], { stdio: 'inherit' });
  console.log(`Requested release launch: ${bundle}. Run make desktop-status to verify the running executable.`);
}
