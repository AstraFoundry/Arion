import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

export function installGitHooks({ env = process.env, rootDir = root, run = execFileSync, exists = existsSync } = {}) {
  if (env.GITHUB_ACTIONS === 'true') {
    console.log('pre-push hook: skipped on GitHub Actions');
    return false;
  }
  if (!exists(path.join(rootDir, '.git'))) throw new Error('pre-push hook: checkout has no .git directory');
  run('git', ['config', 'core.hooksPath', '.githooks'], { cwd: rootDir, stdio: 'inherit' });
  console.log('pre-push hook: core.hooksPath=.githooks');
  return true;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) installGitHooks();
