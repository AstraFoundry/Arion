// Builds and tests isolated native boundaries; never installs a TSF profile.
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
if (process.platform !== 'win32') throw new Error('Windows keyboard checks require native Windows.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
execFileSync(process.execPath, [path.join(root, 'scripts/check-emoji-data.mjs')], { cwd: root, stdio: 'inherit', windowsHide: true });
for (const script of ['build-windows-input-method', 'test-windows-tsf-loader', 'test-windows-ime-registration', 'test-windows-ime-client', 'test-windows-language-bar', 'check-shared-keyboard', 'test-windows-pinyin', 'test-windows-keyboard-module', 'test-windows-tsf-keyboard']) {
  execFileSync(process.execPath, [path.join(root, 'scripts', `${script}.mjs`)], { cwd: root, stdio: 'inherit', windowsHide: true });
}
