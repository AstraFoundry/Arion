import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, renameSync, rmSync } from 'node:fs';
import path from 'node:path';

const module = 'github.com/rhysd/actionlint';
const version = 'v1.7.11';

// Cache the pinned Go tool, rather than resolving a version over the network
// on every check. Initial provisioning still uses Go's checksum verification
// and the caller's explicit GOPROXY/GOSUMDB configuration.
export function checkWorkflows(root, { run = spawnSync, env = process.env, platform = process.platform, arch = process.arch } = {}) {
  const directory = path.join(root, '.workspace', 'tools', 'actionlint', version, `${platform}-${arch}`);
  const name = platform === 'win32' ? 'actionlint.exe' : 'actionlint';
  const binary = path.join(directory, name);
  if (!existsSync(binary)) {
    mkdirSync(directory, { recursive: true });
    const staging = mkdtempSync(path.join(directory, '.install-'));
    try {
      console.log(`Installing actionlint ${version} into the local tool cache (first run only).`);
      const result = run('go', ['install', `${module}/cmd/actionlint@${version}`], {
        cwd: root, env: { ...env, GOWORK: 'off', GOBIN: staging }, stdio: 'inherit',
      });
      if (result.error || result.status !== 0) {
        console.error('actionlint installation failed. Check your terminal network/proxy and GOPROXY configuration, then rerun make workflows-check. The workflow check has not run.');
        if (result.error) console.error(result.error.message);
        return result.status || 1;
      }
      renameSync(path.join(staging, name), binary);
    } finally {
      rmSync(staging, { recursive: true, force: true });
    }
  }
  // A stale or manually replaced tool must not silently weaken this gate.
  const info = run('go', ['version', '-m', binary], { cwd: root, env, encoding: 'utf8' });
  const pinned = info.stdout?.split('\n').some(line => {
    const fields = line.trim().split(/\s+/);
    return fields[0] === 'mod' && fields[1] === module && fields[2] === version;
  });
  if (info.error || info.status !== 0 || !pinned) {
    console.error(`Invalid actionlint cache: expected ${module}@${version}. Remove ${directory} and rerun make workflows-check.`);
    return 1;
  }
  const result = run(binary, ['-shellcheck=', '-pyflakes='], { cwd: root, env, stdio: 'inherit' });
  if (result.error) console.error(result.error.message);
  return result.status ?? 1;
}
