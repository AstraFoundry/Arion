// Derives OS permission copy from the same catalogs used by native views.
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const config = JSON.parse(readFileSync(path.join(desktop, 'desktop.json'), 'utf8'));
const descriptions = {
  NSMicrophoneUsageDescription: 'SystemMicrophoneUsage',
  NSDocumentsFolderUsageDescription: 'SystemDocumentsUsage',
  NSDesktopFolderUsageDescription: 'SystemDesktopUsage',
  NSDownloadsFolderUsageDescription: 'SystemDownloadsUsage',
};
for (const [locale, bundleLanguage] of [['en', 'en'], ['zh-CN', 'zh-Hans']]) {
  const catalog = JSON.parse(readFileSync(path.join(desktop, 'ui/shared/copy/locales', `${locale}.json`), 'utf8'));
  const strings = Object.entries(descriptions).map(([property, key]) => {
    const message = catalog[key];
    if (typeof message !== 'string' || !message.trim()) throw new Error(`Missing ${key} permission copy for ${locale}`);
    if (locale === 'en' && config.macos.infoPlist[property] !== message) {
      throw new Error(`desktop.json ${property} description must match the English catalog`);
    }
    return `${JSON.stringify(property)} = ${JSON.stringify(message)};`;
  });
  const directory = path.join(desktop, 'resources', 'darwin', `${bundleLanguage}.lproj`);
  mkdirSync(directory, { recursive: true });
  writeFileSync(path.join(directory, 'InfoPlist.strings'), `${strings.join('\n')}\n`);
}
console.log('Staged English and Simplified Chinese macOS permission localizations.');
