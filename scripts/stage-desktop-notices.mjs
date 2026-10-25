// Preserve licenses of the pinned Go modules and the model provenance in packages.
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './lib/pinned-file.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const destination = path.join(desktop, 'resources', 'licenses');
rmSync(destination, { recursive: true, force: true });
mkdirSync(destination, { recursive: true });
copyFileSync(path.join(root, 'LICENSE'), path.join(destination, 'Arion-LICENSE'));
copyFileSync(path.join(root, 'models', 'catalog.json'), path.join(destination, 'model-catalog.json'));
const env = { ...process.env, GOWORK: 'off' };
execFileSync('go', ['mod', 'download', 'all'], { cwd: desktop, env, stdio: 'inherit' });
const modules = execFileSync('go', ['list', '-m', '-f', '{{.Path}}\t{{.Version}}\t{{.Dir}}', 'all'], { cwd: desktop, env, encoding: 'utf8' });
const records = [];
for (const row of modules.trim().split('\n')) {
  const [name, version, directory] = row.split('\t');
  if (name.startsWith('arion.local/')) continue;
  const licenses = readdirSync(directory).filter(file => /^(LICENSE|LICENCE|COPYING|NOTICE)(\..*)?$/i.test(file));
  if (!licenses.length) throw new Error(`No license found for ${name}@${version}`);
  const folder = name.replaceAll('/', '_');
  mkdirSync(path.join(destination, folder));
  for (const file of licenses) copyFileSync(path.join(directory, file), path.join(destination, folder, file));
  records.push({ module: name, version, ...(name === 'github.com/egoist/mygo' ? { source: 'third_party/mygo', patch: 'prerelease-channel-v1, url-channel-v1, codesign-metadata-v1, windows-native-transparency-v1' } : {}), licenses: licenses.map(file => `${folder}/${file}`) });
}
writeFileSync(path.join(destination, 'go-modules.json'), `${JSON.stringify(records, null, 2)}\n`);
const native = JSON.parse(readFileSync(path.join(root, 'scripts/native-notices.json'), 'utf8'));
for (const spec of native.files) copyFileSync(pinnedFile(spec), path.join(destination, spec.name));
copyFileSync(path.join(root, 'scripts/native-notices.json'), path.join(destination, 'native-notices.json'));
console.log(`Staged Arion, model provenance and ${records.length} module license records.`);
