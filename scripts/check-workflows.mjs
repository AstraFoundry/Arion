import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
execFileSync('go', ['run', 'github.com/rhysd/actionlint/cmd/actionlint@v1.7.11', '-shellcheck=', '-pyflakes='], { cwd: root, stdio: 'inherit' });
