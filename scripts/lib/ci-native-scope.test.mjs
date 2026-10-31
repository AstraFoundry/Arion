import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { isDocumentationOnlyPath, nativeShellRequired } from '../ci-native-scope.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

test('site, docs, and root READMEs do not require the native shell', () => {
  const files = [
    'apps/site/app/page.tsx',
    'apps/site/content/blog/where-your-dictation-goes/en.md',
    'docs/operations/README.md',
    'README.md',
    'README.zh-CN.md',
    'README.ja.md',
  ];
  for (const file of files) assert.equal(isDocumentationOnlyPath(file), true, file);
  assert.equal(nativeShellRequired(files), false);
  assert.equal(nativeShellRequired(['docs/operations/deployment.md']), false);
});

test('desktop, IME, Go, scripts, and workflow paths keep the native shell', () => {
  const files = [
    'apps/desktop/ui/app/app.go',
    'apps/desktop/ime/shared/keyboard-data/arion_pinyin.schema.yaml',
    'packages/core/go.mod',
    'models/go.mod',
    'services/sync/go.mod',
    'go.work',
    'scripts/check-windows-keyboard-ci.ps1',
    'scripts/build.mjs',
    'third_party/mygo/go.mod',
    '.github/workflows/ci.yml',
    'pnpm-lock.yaml',
    'specs/001-desktop-input.md',
    'AGENTS.md',
    'apps/site',
    'docs',
  ];
  for (const file of files) {
    assert.equal(isDocumentationOnlyPath(file), false, file);
    assert.equal(nativeShellRequired(['docs/operations/README.md', file]), true, file);
  }
});

test('an empty diff or an unsafe path fails closed and runs native', () => {
  assert.equal(nativeShellRequired([]), true);
  assert.equal(nativeShellRequired(null), true);
  for (const file of ['', '/docs/README.md', 'docs/../apps/desktop/ui/app/app.go', '..\\apps\\desktop\\go.mod']) {
    assert.equal(isDocumentationOnlyPath(file), false, file);
    assert.equal(nativeShellRequired([file]), true, file);
  }
});

test('CI pins one Go toolchain and keeps release concurrency', () => {
  const ci = readFileSync(path.join(root, '.github/workflows/ci.yml'), 'utf8');
  const release = readFileSync(path.join(root, '.github/workflows/release.yml'), 'utf8');
  const signed = readFileSync(path.join(root, '.github/workflows/signed-macos.yml'), 'utf8');
  assert.deepEqual([...ci.matchAll(/go-version: '([^']+)'/g)].map(match => match[1]), ['1.27.3', '1.27.3', '1.27.3']);
  assert.match(ci, /cancel-in-progress: \$\{\{ github\.event_name == 'pull_request' \}\}/);
  assert.match(ci, /check-windows-keyboard-ci\.ps1/);
  assert.equal([...ci.matchAll(/github\.event_name != 'pull_request' \|\| inputs\.release/g)].length, 3);
  assert.match(ci, /github\.event_name == 'workflow_call' \|\| github\.event_name == 'workflow_dispatch' \|\| github\.ref_type == 'tag'/);
  assert.match(ci, /native-restore/);
  assert.match(ci, /go-restore/);
  assert.match(release, /uses: \.\/\.github\/workflows\/ci\.yml/);
  assert.match(release, /group: release-\$\{\{ github\.ref \}\}/);
  assert.match(release, /cancel-in-progress: false/);
  assert.match(signed, /uses: \.\/\.github\/workflows\/ci\.yml/);
  assert.match(signed, /group: signed-macos-\$\{\{ github\.ref \}\}/);
  assert.match(signed, /cancel-in-progress: false/);
  assert.doesNotMatch(ci, /paths-ignore:/);
});
