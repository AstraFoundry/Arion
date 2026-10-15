// Stages the local rewrite worker into the desktop platform resources:
// the pinned ONNX Runtime GenAI and ONNX Runtime libraries (verified by
// SHA-256) and an `arion-llm` build without cgo. `mygo build` packages and
// signs them with the app. The worker runs in its own process, so its ONNX
// Runtime never shares an address space with the speech runtime's.
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync, readFileSync, rmSync, chmodSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './lib/pinned-file.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const pins = JSON.parse(readFileSync(path.join(root, 'scripts', 'llm-runtime.json'), 'utf8'));
const run = (command, args, options = {}) => execFileSync(command, args, { stdio: 'inherit', ...options });

const platform = process.argv[2] ?? `${process.platform === 'win32' ? 'windows' : process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}`;
const spec = pins.platforms[platform];
if (!spec) {
  // No pinned runtime: the app reports the capability as unsupported.
  console.log(`No rewrite runtime is pinned for ${platform}; skipped.`);
  process.exit(0);
}
const destination = path.join(desktop, 'resources', platform, 'llm');
rmSync(destination, { recursive: true, force: true });
mkdirSync(destination, { recursive: true });
for (const archive of spec.archives) {
  const file = pinnedFile(archive);
  const unpacked = `${file}.d`;
  rmSync(unpacked, { recursive: true, force: true });
  mkdirSync(unpacked);
  run('tar', ['-xzf', file, '-C', unpacked, ...Object.keys(archive.members)]);
  for (const [member, name] of Object.entries(archive.members)) {
    const target = path.join(destination, name);
    mkdirSync(path.dirname(target), { recursive: true });
    copyFileSync(path.join(unpacked, member), target);
    chmodSync(target, name.endsWith('.dylib') ? 0o755 : 0o644);
  }
  rmSync(unpacked, { recursive: true, force: true });
}
const [goos, goarch] = platform.split('-');
const worker = path.join(destination, goos === 'windows' ? 'arion-llm.exe' : 'arion-llm');
run('go', ['build', '-trimpath', '-o', worker, './cmd/arion-llm'], { cwd: desktop, env: { ...process.env, CGO_ENABLED: '0', GOOS: goos, GOARCH: goarch } });
console.log(`${path.relative(root, destination)}: ${Object.values(spec.archives.flatMap(a => Object.values(a.members))).join(', ')}, ${path.basename(worker)}`);
