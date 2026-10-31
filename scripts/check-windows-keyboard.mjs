// Builds and tests isolated native boundaries; never installs a TSF profile.
import { execFileSync } from 'node:child_process';
import { performance } from 'node:perf_hooks';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
if (process.platform !== 'win32') throw new Error('Windows keyboard checks require native Windows.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const scripts = ['check-emoji-data', 'build-windows-input-method', 'test-windows-tsf-loader', 'test-windows-ime-registration', 'test-windows-ime-client', 'test-windows-language-bar', 'check-shared-keyboard', 'test-windows-pinyin', 'test-windows-keyboard-module', 'test-windows-keyboard-host', 'test-windows-tsf-keyboard'];
for (const script of scripts) {
  const started = performance.now();
  execFileSync(process.execPath, [path.join(root, 'scripts', `${script}.mjs`)], { cwd: root, stdio: 'inherit', windowsHide: true });
  console.log(`Windows keyboard step ${script}: ${((performance.now() - started) / 1000).toFixed(3)} seconds`);
}
