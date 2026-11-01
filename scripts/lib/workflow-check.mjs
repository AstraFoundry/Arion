import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, renameSync, rmSync } from 'node:fs';
import path from 'node:path';

const version = 'v1.7.11';
// Release archives and extracted executables are independently pinned.
const releases = {
  'linux-x64': { archive: '900919a84f2229bac68ca9cd4103ea297abc35e9689ebb842c6e34a3d1b01b0a', binary: '1ef54b3443db3e2c2ef3b82c565c328ce1d76420ae13e0df3676f936dfcdb77c' },
  'linux-arm64': { archive: '21bc0dfb57a913fe175298c2a9e906ee630f747cb66d0a934d0d4b69f4ee1235', binary: '58139720ed7484ca3f0bcdd7b249709f79db7adfc6695418c36b27aa8b04c8b7' },
  'darwin-x64': { archive: '17ffc17fed8f0258ef6ad4aed932d3272464c7ef7d64e1cb0d65aa97c9752107', binary: '0c67c5c071d1d0a60285c13a592e11dec3aad8de8f30a8e789000ec7f06bbad8' },
  'darwin-arm64': { archive: 'a21ba7366d8329e7223faee0ed69eb13da27fe8acabb356bb7eb0b7f1e1cb6d8', binary: '346aeea519e83f28da729140c0e3c07d133e0a148f72de77ed3410c76287e2a1' },
  'win32-x64': { archive: '5414b7124a91f4b5abee62e5c9d84802237734f8d15b9b7032732a32c3ebffa3', binary: '1aeec711cbd8b584aaf99a00126ff346aaf74e7168faf8aed2a976370bc97a72' },
  'win32-arm64': { archive: '4900eb60e084604850935705306305f0bb0ca532e07c2b4ae642917142aba75d', binary: '96f3ba67b622fe1b1c130f1cf203264ef862202f368cec2eedb07c6df3f2c218' },
};
const sha256 = file => createHash('sha256').update(readFileSync(file)).digest('hex');

export function checkWorkflows(root, { run = spawnSync, env = process.env, platform = process.platform, arch = process.arch, artifacts = releases } = {}) {
  const artifact = artifacts[`${platform}-${arch}`];
  if (!artifact) throw new Error(`Unsupported actionlint platform: ${platform}-${arch}`);
  const directory = path.join(root, '.workspace', 'tools', 'actionlint', version, 'prebuilt', `${platform}-${arch}`);
  const name = platform === 'win32' ? 'actionlint.exe' : 'actionlint';
  const binary = path.join(directory, name);
  if (!existsSync(binary)) {
    mkdirSync(directory, { recursive: true });
    const staging = mkdtempSync(path.join(directory, '.install-'));
    try {
      const releaseOS = platform === 'win32' ? 'windows' : platform;
      const releaseArch = arch === 'x64' ? 'amd64' : arch;
      const file = `actionlint_${version.slice(1)}_${releaseOS}_${releaseArch}.${platform === 'win32' ? 'zip' : 'tar.gz'}`;
      const archive = path.join(staging, file);
      console.log(`Downloading prebuilt actionlint ${version}.`);
      const download = run('curl', ['--fail', '--location', '--silent', '--show-error', '--output', archive,
        `https://github.com/rhysd/actionlint/releases/download/${version}/${file}`], { cwd: root, env, stdio: 'inherit' });
      if (download.error || download.status !== 0) return download.status || 1;
      if (sha256(archive) !== artifact.archive) throw new Error('actionlint archive SHA-256 mismatch.');
      const extract = run('tar', ['-xf', archive, '-C', staging, name], { cwd: root, env, stdio: 'inherit' });
      if (extract.error || extract.status !== 0) return extract.status || 1;
      if (sha256(path.join(staging, name)) !== artifact.binary) throw new Error('actionlint executable SHA-256 mismatch.');
      renameSync(path.join(staging, name), binary);
    } finally {
      rmSync(staging, { recursive: true, force: true });
    }
  }
  if (sha256(binary) !== artifact.binary) throw new Error(`Invalid actionlint cache: remove ${directory} and rerun make workflows-check.`);
  const result = run(binary, ['-shellcheck=', '-pyflakes='], { cwd: root, env, stdio: 'inherit' });
  if (result.error) console.error(result.error.message);
  return result.status ?? 1;
}
