// One resource preparation path for local packages and clean CI runners.
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { requireDesktopToolchain } from './lib/desktop-toolchain.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
requireDesktopToolchain();
for (const script of ['stage-desktop-localizations', 'stage-asr-runtime', 'stage-llm-runtime', 'stage-keyboard-ranker', 'build-input-method', 'build-windows-input-method', 'stage-desktop-notices']) {
  execFileSync(process.execPath, [path.join(root, 'scripts', `${script}.mjs`)], { cwd: root, stdio: 'inherit' });
}
