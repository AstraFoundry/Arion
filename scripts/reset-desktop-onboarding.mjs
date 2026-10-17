// Replay first-run setup without removing the rest of the user's profile.
import { copyFileSync, constants, readFileSync, renameSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import path from 'node:path';
import { randomUUID } from 'node:crypto';

if (process.argv.length > 2) throw new Error('Usage: node scripts/reset-desktop-onboarding.mjs');
if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Desktop reset requires macOS or Windows');

const base = process.platform === 'darwin'
  ? path.join(homedir(), 'Library', 'Application Support')
  : process.env.APPDATA;
if (!base) throw new Error('APPDATA is required to locate desktop preferences');
const directory = process.env.ARION_DATA_DIR || path.join(base, 'Arion');
const preferencesPath = path.join(directory, 'preferences.json');
console.log('Arion must be fully quit before resetting onboarding.');
let source;
try {
  source = readFileSync(preferencesPath, 'utf8');
} catch (error) {
  if (error.code !== 'ENOENT') throw error;
  console.log(`No preferences at ${preferencesPath}; onboarding will appear on the next launch.`);
  process.exit(0);
}
const preferences = JSON.parse(source);
if (!preferences || typeof preferences !== 'object' || Array.isArray(preferences)) {
  throw new Error(`Expected a preferences object at ${preferencesPath}`);
}
if (preferences.onboardingCompleted === false) {
  console.log('Onboarding is already reset. Restart Arion to begin setup.');
  process.exit(0);
}
const backupPath = `${preferencesPath}.${randomUUID()}.bak`;
copyFileSync(preferencesPath, backupPath, constants.COPYFILE_EXCL);
preferences.onboardingCompleted = false;
const temporaryPath = `${preferencesPath}.${randomUUID()}.tmp`;
writeFileSync(temporaryPath, `${JSON.stringify(preferences, null, 2)}\n`, { mode: 0o600, flag: 'wx' });
renameSync(temporaryPath, preferencesPath);
console.log(`Preferences backed up to ${backupPath}`);
console.log('Onboarding reset. Restart Arion to begin setup.');
