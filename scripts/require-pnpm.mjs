import { installGitHooks } from './install-git-hooks.mjs';

const agent = process.env.npm_config_user_agent ?? '';
if (!agent.startsWith('pnpm/')) {
  console.error('Use the pnpm version pinned in package.json.');
  process.exitCode = 1;
} else installGitHooks();
