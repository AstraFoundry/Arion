import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
execFileSync(process.execPath, ['--test', path.join(root, 'scripts/lib/keyboard-english.test.mjs')], { cwd: root, stdio: 'inherit', windowsHide: true });
for (const script of ['test-english-lexicon', 'test-shared-pinyin', 'test-shared-keyboard-learning', 'test-keyboard-spelling']) {
  execFileSync(process.execPath, [path.join(root, 'scripts', `${script}.mjs`)], { cwd: root, stdio: 'inherit', windowsHide: true });
}
