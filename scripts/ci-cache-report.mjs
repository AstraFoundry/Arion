import path from 'node:path';
import { fileURLToPath } from 'node:url';

// One grep target for the next Actions log. exact=false covers both a
// restore-key prefix hit and a cold start; the cache action's own line
// distinguishes "Cache hit for restore-key" from "Cache not found".
export function cacheReport({ name, exact, key = '', restore = '', outcome }) {
  if (!/^[a-z0-9-]+$/.test(name)) throw new Error(`Invalid cache name: ${name}`);
  let state = exact === 'true' ? 'exact-hit' : exact === 'false' ? 'no-exact-hit' : exact === '' || exact === undefined ? 'disabled' : null;
  if (!state) throw new Error(`Invalid cache exact flag: ${exact}`);
  if (outcome === 'success' && state === 'disabled') state = 'no-exact-hit';
  else if (outcome && outcome !== 'success') state = 'disabled';
  return [`ci-cache ${name}: ${state}`, key && `key=${key}`, restore && `restore=${restore}`].filter(Boolean).join(' ');
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [name, exact = '', key = '', restore = '', outcome] = process.argv.slice(2);
  if (!name) throw new Error('ci-cache-report.mjs requires a cache name.');
  console.log(cacheReport({ name, exact, key, restore, outcome }));
}
