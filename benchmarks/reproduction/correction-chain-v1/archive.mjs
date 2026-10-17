// Preserve original benchmark bytes; weights and runtime environments are excluded.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hash } from './report.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const destination = path.join(root, 'benchmarks/runs/correction-chain-v1');
function files(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(directory, entry.name);
    return entry.isDirectory() ? files(full) : entry.isFile() ? [full] : [];
  });
}
export function verify(manifestPath) {
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  for (const item of manifest.files) {
    const file = path.resolve(path.dirname(manifestPath), item.path);
    if (hash(file) !== item.sha256) throw new Error(`Archive digest mismatch: ${item.path}`);
  }
  return { verifiedFiles: manifest.files.length };
}
function archive() {
  fs.mkdirSync(destination, { recursive: false });
  const source = path.join(root, '.workspace/correction-chain');
  const directories = ['dev-run', 'dev-run-v2', 'dev-run-v3', 'dev-run-v4', 'test-run', 'preflight-flanec', 'preflight-rewrite', 'provision'];
  for (const name of directories) fs.cpSync(path.join(source, name), path.join(destination, name), { recursive: true, errorOnExist: true, force: false });
  for (const name of ['dev-run.log', 'dev-run-v2.log', 'dev-run-v3.log', 'dev-run-v4.log', 'test-run.log', 'fixtures.log']) fs.copyFileSync(path.join(source, name), path.join(destination, name), fs.constants.COPYFILE_EXCL);
  for (const name of ['bart4csc', 'flanec']) {
    const target = path.join(destination, 'provision', name);
    fs.mkdirSync(target);
    for (const file of ['manifest.json', 'upstream.json', 'README.md', 'config.json']) fs.copyFileSync(path.join(source, 'models', name, file), path.join(target, file));
  }
  const fixtureRoot = path.join(root, 'benchmarks/corpora/correction-chain-v1');
  const resultRoot = path.join(root, 'benchmarks/results/correction-chain-v1');
  const entries = [...files(destination), ...files(fixtureRoot), ...files(resultRoot), ...files(path.join(root, 'benchmarks/reproduction/correction-chain-v1')).filter(f => !f.includes('/__pycache__/'))].map(file => ({ path: path.relative(destination, file), bytes: fs.statSync(file).size, sha256: hash(file) }));
  const manifestPath = path.join(destination, 'manifest.json');
  fs.writeFileSync(manifestPath, JSON.stringify({ version: 1, kind: 'correction-chain-archive-v1', files: entries }, null, 2) + '\n', { flag: 'wx' });
  return verify(manifestPath);
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  console.log(JSON.stringify(process.argv[2] === 'verify' ? verify(path.join(destination, 'manifest.json')) : archive()));
}
