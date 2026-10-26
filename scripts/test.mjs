import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
for (const [args, cwd] of [
  [['--test', 'scripts/lib/reset-desktop.test.mjs', 'scripts/lib/markdown-anchors.test.mjs', 'scripts/lib/release.test.mjs', 'scripts/lib/workflow-check.test.mjs'], root],
  [['scripts/check-go.mjs', 'core'], root],
  [['scripts/check-go.mjs', 'sync'], root],
  [['scripts/check-go.mjs', 'updates'], root],
  [['scripts/check-i18n.mjs'], root],
  [['--test'], path.join(root, 'tools/evaluation')],
]) {
  const result = spawnSync(process.execPath, args, { cwd, stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}
