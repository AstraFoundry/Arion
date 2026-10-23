import {execFileSync} from 'node:child_process';
import {mkdtempSync, rmSync} from 'node:fs';
import {tmpdir} from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
if (process.platform !== 'darwin') process.exit(0);
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const temporary = mkdtempSync(path.join(tmpdir(), 'arion-source-maintenance-'));
try {
  const binary = path.join(temporary, 'source-maintenance-test');
  execFileSync('xcrun', ['clang', '-fobjc-arc', '-framework', 'Foundation', '-framework', 'Carbon',
    path.join(root, 'apps/desktop/ime/macos/tests/source_maintenance_test.m'), '-o', binary], {stdio: 'inherit'});
  execFileSync(binary, [], {stdio: 'inherit'});
} finally { rmSync(temporary, {recursive: true, force: true}); }
