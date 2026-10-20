// This entry point is destructive only after argument/path checks and system preflight.
import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import { homedir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { clearProfile, resetOptions, resetPlan } from './lib/reset-desktop.mjs';
import { prepareSystemReset } from './lib/reset-desktop-system.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const options = resetOptions(process.argv.slice(2));
if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Desktop reset requires macOS or Windows');
const home = homedir();
const configBase = process.platform === 'darwin' ? path.join(home, 'Library/Application Support') : process.env.APPDATA;
if (!configBase) throw new Error('APPDATA is required to locate the desktop profile');
const directory = process.env.ARION_DATA_DIR || path.join(configBase, 'Arion');
const plan = resetPlan(directory, options, { home, repository: root, configBase });
console.log(JSON.stringify(plan, null, 2));
if (options.dryRun) {
  console.log('Preview only; no files, credentials, permissions or input sources were changed.');
} else {
  if (process.env.ARION_DATA_DIR && Object.values(options.system).some(Boolean)) {
    throw new Error('ARION_DATA_DIR does not isolate OS state. Append keep-system to reset an isolated profile.');
  }
  const processes = process.platform === 'darwin'
    ? execFileSync('/bin/ps', ['-axo', 'comm='], { encoding: 'utf8' }).split('\n').map(line => path.basename(line.trim()))
    : execFileSync('tasklist', ['/FO', 'CSV', '/NH'], { encoding: 'utf8' }).split('\n').map(line => line.match(/^"([^"]+)"/)?.[1]);
  if (processes.some(name => /^(Arion|arion|arion-llm)(\.exe)?$/.test(name ?? ''))) {
    throw new Error('Fully quit Arion and its inference workers before resetting.');
  }
  // The input-method process is terminated by the native system reset. If it
  // is retained, the user must stop it too so it cannot recreate profile data.
  if (!options.system.inputMethod && processes.some(name => name === 'ArionInput')) {
    throw new Error('Stop ArionInput before resetting with keep-input-method or keep-system.');
  }
  const envFile = path.join(root, '.env');
  if (existsSync(envFile)) process.loadEnvFile(envFile);
  const system = prepareSystemReset(root, options.system);
  try {
    system.check();
    system.execute();
    clearProfile(plan);
    console.log('Reset complete. Start Arion again to use the remaining profile data and defaults.');
  } finally {
    system.cleanup();
  }
}
