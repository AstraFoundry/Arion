// Bound main's source-keyed native caches without touching setup-action caches.
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function obsoleteNativeCaches(caches, keep = 2) {
  if (!Number.isInteger(keep) || keep < 1) throw new Error('Cache retention must be positive.');
  const groups = new Map();
  for (const cache of caches) {
    if (cache.ref !== 'refs/heads/main') continue;
    const family = cache.key.match(/^(native|standard-user-go)-v\d+-(darwin|win32)-(arm64|x64)-([a-zA-Z0-9]+)-/);
    if (!family) continue;
    const group = family.slice(1).join('-');
    if (!groups.has(group)) groups.set(group, []);
    groups.get(group).push(cache);
  }
  return [...groups.values()].flatMap(group => group.sort((a, b) => b.created_at.localeCompare(a.created_at) || b.id - a.id).slice(keep));
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (process.env.GITHUB_REF !== 'refs/heads/main' || process.env.GITHUB_EVENT_NAME !== 'push') {
    throw new Error('Native cache pruning is restricted to main pushes.');
  }
  const pages = JSON.parse(execFileSync('gh', ['api', '--paginate', '--slurp', 'repos/{owner}/{repo}/actions/caches?per_page=100'], { encoding: 'utf8' }));
  for (const cache of obsoleteNativeCaches(pages.flatMap(page => page.actions_caches))) {
    execFileSync('gh', ['api', '--method', 'DELETE', `repos/{owner}/{repo}/actions/caches/${cache.id}`], { stdio: 'inherit' });
    console.log(`Removed superseded native cache ${cache.id}: ${cache.key} (${cache.size_in_bytes} bytes)`);
  }
}
