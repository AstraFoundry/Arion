import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
for (const [args, cwd] of [
  [['scripts/verify-public-fixtures.mjs'], root],
  [['--test', 'scripts/lib/catalog-evaluation.test.mjs', 'scripts/lib/reset-desktop.test.mjs', 'scripts/lib/markdown-anchors.test.mjs', 'scripts/lib/release.test.mjs', 'scripts/lib/workflow-check.test.mjs', 'scripts/lib/doc-consistency.test.mjs', 'scripts/lib/center-profile.test.mjs', 'scripts/lib/desktop-signing.test.mjs', 'scripts/lib/desktop-development.test.mjs', 'scripts/lib/ime-provisioning.test.mjs', 'scripts/lib/macos-distribution.test.mjs', 'scripts/lib/release-policy.test.mjs', 'scripts/lib/windows-pe.test.mjs'], root],
  [['scripts/check-go.mjs', 'core'], root],
  [['scripts/check-go.mjs', 'sync'], root],
  [['scripts/check-go.mjs', 'updates'], root],
  [['scripts/check-i18n.mjs'], root],
  [['--test'], path.join(root, 'tools/driver')],
]) {
  const result = spawnSync(process.execPath, args, { cwd, stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}
