import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { changedFiles, isDocumentationOnlyPath, nativeShellRequired, portableTestsRequired, siteChecksRequired } from '../ci-native-scope.mjs';

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
  assert.equal([...ci.matchAll(/github\.event_name != 'pull_request' \|\| inputs\.release/g)].length, 0);
  assert.equal([...ci.matchAll(/^ {8}if: inputs\.release$/gm)].length, 3);
  assert.match(ci, /collect \$\{\{ matrix\.platform \}\} \.workspace\/release\/\$\{\{ matrix\.platform \}\} --signed/);
  assert.doesNotMatch(ci, /--unsigned/);
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


test('only documentation skips portable tests; site changes and unknown diffs keep them', () => {
  assert.equal(portableTestsRequired(['docs/operations/README.md', 'README.md']), false);
  for (const files of [[], null, ['apps/site/app/page.tsx'], ['docs/README.md', 'scripts/test.mjs'], ['specs/README.md']]) {
    assert.equal(portableTestsRequired(files), true);
  }
});

test('desktop-only changes skip the site while preserving portable and native checks', () => {
  const files = ['apps/desktop/infrastructure/audio/device.go', 'packages/core/domain/ports/diagnostics.go',
    'third_party/malgo/miniaudio.h', 'docs/backend/logging.md', 'specs/001-desktop-input.md'];
  assert.equal(siteChecksRequired(files), false);
  assert.equal(portableTestsRequired(files), true);
  assert.equal(nativeShellRequired(files), true);
  assert.equal(siteChecksRequired(['docs/README.md', 'README.md']), false);
  assert.equal(siteChecksRequired(['scripts/check-windows-keyboard-ci.ps1', 'services/sync/go.mod']), false);
});

test('site source, content, assets and shared build configuration require site checks', () => {
  for (const file of ['apps/site/app/page.tsx', 'apps/site/content/blog/post/en.md',
    'apps/site/public/logo.png', 'apps/site/.env.production', 'apps/site/package.json',
    'apps\\site\\next.config.ts', 'apps/site', 'package.json', 'pnpm-lock.yaml',
    'pnpm-workspace.yaml', '.npmrc', '.pnpmfile.cjs', '.nvmrc', '.node-version',
    'patches/next.patch', 'scripts/require-pnpm.mjs', 'scripts/build.mjs', 'scripts/update-blog-authors.mjs',
    'scripts/ci-native-scope.mjs', 'scripts/ci-required-checks.mjs',
    '.github/workflows/ci.yml', '.github/actions/setup/action.yml']) {
    assert.equal(siteChecksRequired(['apps/desktop/go.mod', file]), true, file);
  }
  for (const files of [[], null, [''], [null], ['/apps/site/app/page.tsx'], ['docs/../apps/site/page.tsx']]) {
    assert.equal(siteChecksRequired(files), true, JSON.stringify(files));
  }
});

test('deleting or moving a site file out of the site still requires its checks', t => {
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-site-scope-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const git = (...args) => execFileSync('git', args, { cwd: directory, encoding: 'utf8' }).trim();
  git('init', '-q');
  git('config', 'user.name', 'scope');
  git('config', 'user.email', 'scope@example.test');
  mkdirSync(path.join(directory, 'apps/site'), { recursive: true });
  writeFileSync(path.join(directory, 'apps/site/page.tsx'), 'export default 1;\n');
  git('add', '.');
  git('commit', '-qm', 'base');
  const base = git('rev-parse', 'HEAD');
  git('mv', 'apps/site/page.tsx', 'moved.tsx');
  git('commit', '-qm', 'move site source');
  const files = changedFiles(base, directory);
  assert.deepEqual(files, ['apps/site/page.tsx', 'moved.tsx']);
  assert.equal(siteChecksRequired(files), true);
});

test('CI gates the site independently and passes its actual outcome to the required check', () => {
  const ci = readFileSync(path.join(root, '.github/workflows/ci.yml'), 'utf8');
  assert.match(ci, /id: site-check\n\s+if: github\.event_name != 'pull_request' \|\| needs\.scope\.outputs\.site != 'false'/);
  assert.match(ci, /site: \$\{\{ steps\.decision\.outputs\.site \}\}/);
  assert.match(ci, /site-result: \$\{\{ steps\.site-check\.outcome \}\}/);
  assert.match(ci, /SITE_RESULT: \$\{\{ needs\.portable\.outputs\.site-result \}\}/);
});

test('main uses a unique concurrency group and never cancels on portable failure', () => {
  const ci = readFileSync(path.join(root, '.github/workflows/ci.yml'), 'utf8');
  assert.match(ci, /github\.event_name == 'pull_request' && github\.ref \|\| github\.run_id/);
  assert.match(ci, /if: failure\(\) && github\.event_name == 'pull_request'/);
});
