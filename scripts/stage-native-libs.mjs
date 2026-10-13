// Copies the pinned sherpa-onnx/ONNX Runtime shared libraries from the Go
// module cache into the desktop platform resources, where `mygo build`
// packages and signs them. Libraries are build artifacts and stay ignored.
import { execFileSync } from 'node:child_process';
import { chmodSync, copyFileSync, mkdirSync, readdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const moduleDir = name => execFileSync('go', ['list', '-m', '-f', '{{.Dir}}', name], { cwd: desktop, encoding: 'utf8' }).trim();
const targets = [
  { module: 'github.com/k2-fsa/sherpa-onnx-go-macos', from: 'aarch64-apple-darwin', to: 'darwin-arm64/lib', keep: /\.dylib$/ },
  { module: 'github.com/k2-fsa/sherpa-onnx-go-macos', from: 'x86_64-apple-darwin', to: 'darwin-amd64/lib', keep: /\.dylib$/ },
  // Windows loads DLLs from the executable's directory.
  { module: 'github.com/k2-fsa/sherpa-onnx-go-windows', from: 'x86_64-pc-windows-gnu', to: 'windows-amd64', keep: /\.dll$/ },
];
for (const target of targets) {
  let source;
  try { source = path.join(moduleDir(target.module), 'lib', target.from); }
  catch { console.error(`missing module ${target.module}; run go mod download in apps/desktop`); process.exit(1); }
  const destination = path.join(desktop, 'resources', target.to);
  mkdirSync(destination, { recursive: true });
  const files = readdirSync(source).filter(name => target.keep.test(name) && !name.includes('cxx'));
  if (files.length === 0) { console.error(`no libraries in ${source}`); process.exit(1); }
  for (const name of files) {
    // Module cache files are read-only: replace, then make the copy writable.
    const target = path.join(destination, name);
    rmSync(target, { force: true });
    copyFileSync(path.join(source, name), target);
    chmodSync(target, 0o644);
  }
  console.log(`${target.to}: ${files.join(', ')}`);
}
