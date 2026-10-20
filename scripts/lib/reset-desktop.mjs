import { lstatSync, readdirSync, rmSync } from 'node:fs';
import path from 'node:path';

export const preservedEntries = {
  'keep-models': ['models'],
  'keep-preferences': ['preferences.json', 'keyboard.json'],
  'keep-vocabulary': ['vocabulary.bin', 'vocabulary-key.dpapi'],
  'keep-keyboard': ['keyboard'],
  'keep-usage': ['usage.json'],
  'keep-credentials': [],
  'keep-permissions': [],
  'keep-input-method': [],
  'keep-system': ['vocabulary-key.dpapi'],
};

export function resetOptions(args) {
  const allowed = new Set([...Object.keys(preservedEntries), '--dry-run']);
  for (const arg of args) if (!allowed.has(arg)) throw new Error(`Unknown reset option: ${arg}`);
  const keep = new Set(args);
  const system = name => !keep.has('keep-system') && !keep.has(name);
  return {
    keep,
    dryRun: keep.has('--dry-run'),
    system: {
      credentials: system('keep-credentials'),
      vocabulary: system('keep-vocabulary'),
      keyboard: system('keep-keyboard'),
      permissions: system('keep-permissions'),
      inputMethod: system('keep-input-method'),
    },
  };
}

export function resetPlan(directory, options, { home, repository, configBase }) {
  if (!path.isAbsolute(directory)) throw new Error('ARION_DATA_DIR must be absolute for reset');
  directory = path.resolve(directory);
  const contains = (parent, child) => {
    const relative = path.relative(parent, child);
    return relative === '' || (!relative.startsWith(`..${path.sep}`) && relative !== '..' && !path.isAbsolute(relative));
  };
  if ([home, repository, configBase].some(protectedPath => contains(directory, path.resolve(protectedPath))) ||
      directory.split(path.sep).filter(Boolean).length < 3) {
    throw new Error(`Refusing unsafe reset directory: ${directory}`);
  }
  // Refuse symlinked ancestors as well as a symlinked profile root. Child
  // symlinks are removed as links; rm never traverses their destinations.
  for (let current = directory; current !== path.dirname(current); current = path.dirname(current)) {
    let info;
    try { info = lstatSync(current); } catch (error) {
      if (error.code === 'ENOENT') continue;
      throw error;
    }
    if (info.isSymbolicLink()) throw new Error(`Refusing symlinked reset directory: ${current}`);
    if (!info.isDirectory()) throw new Error(`Expected a directory: ${current}`);
  }
  const preserved = new Set([...options.keep].flatMap(option => preservedEntries[option] ?? []));
  let entries;
  try { entries = readdirSync(directory).sort(); } catch (error) {
    if (error.code !== 'ENOENT') throw error;
    entries = [];
  }
  return { directory, remove: entries.filter(entry => !preserved.has(entry)), keep: entries.filter(entry => preserved.has(entry)), system: options.system };
}

export function clearProfile(plan) {
  for (const entry of plan.remove) rmSync(path.join(plan.directory, entry), { recursive: true, force: true });
}
