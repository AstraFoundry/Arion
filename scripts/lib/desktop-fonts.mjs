import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import path from 'node:path';

export function desktopFonts(root) {
  const directory = path.join(root, 'apps/desktop/ui/shared/typography/fonts');
  const manifest = JSON.parse(readFileSync(path.join(directory, 'manifest.json'), 'utf8'));
  for (const spec of manifest.files) {
    const data = readFileSync(path.join(directory, spec.name));
    if (data.length !== spec.sizeBytes || createHash('sha256').update(data).digest('hex') !== spec.sha256) {
      throw new Error('Bundled font digest mismatch: ' + spec.name);
    }
  }
  return { directory, manifest };
}

export function verifyEmbeddedFonts(root, application) {
  const { directory, manifest } = desktopFonts(root);
  const app = readFileSync(application);
  const notices = path.join(path.dirname(application), 'licenses');
  if (!readFileSync(path.join(notices, 'SourceHanSans-OFL.txt')).equals(readFileSync(path.join(directory, manifest.license)))) {
    throw new Error('Packaged Source Han license mismatch');
  }
  if (!readFileSync(path.join(notices, 'source-han-sans.json')).equals(readFileSync(path.join(directory, 'manifest.json')))) {
    throw new Error('Packaged Source Han provenance mismatch');
  }
  for (const spec of manifest.files.filter(file => file.name.endsWith('.otf'))) {
    const data = readFileSync(path.join(directory, spec.name));
    if (!app.includes(data)) throw new Error('App is missing bundled font: ' + spec.name);
  }
}
