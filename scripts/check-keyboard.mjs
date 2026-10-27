// Prepare the pinned public runtime, then exercise actual native typing and
// encrypted storage with isolated fixtures. No source registration or Keychain.
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') throw new Error('Keyboard checks require macOS.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
execFileSync(process.execPath, [path.join(root, 'scripts/check-emoji-data.mjs')], { cwd: root, stdio: 'inherit' });
execFileSync('python3', [path.join(root, 'apps/desktop/ime/shared/emoji_generate.py'), '--check'], { cwd: root, stdio: 'inherit' });
for (const script of ['stage-keyboard-runtime', 'check-shared-keyboard', 'test-input-method', 'test-input-source-maintenance', 'test-keyboard-learning', 'test-keyboard-input']) {
  execFileSync(process.execPath, [path.join(root, 'scripts', `${script}.mjs`)], { cwd: root, stdio: 'inherit' });
}
