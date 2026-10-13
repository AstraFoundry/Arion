// Derives OS permission copy from the same catalogs used by native views.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const config = JSON.parse(readFileSync(path.join(desktop, 'mygo.json'), 'utf8'));
for (const [locale, bundleLanguage] of [['en', 'en'], ['zh-CN', 'zh-Hans']]) {
  const catalog = JSON.parse(readFileSync(path.join(desktop, 'ui/shared/copy/locales', `${locale}.json`), 'utf8'));
  const message = catalog.SystemMicrophoneUsage;
  if (typeof message !== 'string' || !message.trim()) throw new Error(`Missing permission copy for ${locale}`);
  if (locale === 'en' && config.macos.infoPlist.NSMicrophoneUsageDescription !== message) {
    throw new Error('mygo.json microphone description must match the English catalog');
  }
  const directory = path.join(desktop, 'resources', 'darwin', `${bundleLanguage}.lproj`);
  mkdirSync(directory, { recursive: true });
  writeFileSync(path.join(directory, 'InfoPlist.strings'), `"NSMicrophoneUsageDescription" = ${JSON.stringify(message)};\n`);
}
console.log('Staged English and Simplified Chinese macOS permission localizations.');
