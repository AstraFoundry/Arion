import { spawnSync } from 'node:child_process';
import { readFileSync } from 'node:fs';

const nodeMajor = readFileSync(new URL('../../.nvmrc', import.meta.url), 'utf8').trim();
const tools = [
  ['go', ['version'], 'Install the Go toolchain documented in docs/operations/local-dev.md.'],
  ['cargo', ['--version'], 'Install Rust/Cargo from https://rustup.rs, then reopen the terminal or source "$HOME/.cargo/env".'],
  ['rustc', ['--version'], 'Install the native Rust toolchain with rustup; Cargo alone is insufficient.'],
  ['cmake', ['--version'], 'Install CMake and add its executable directory to PATH.'],
];

export function desktopToolchainProblems({ nodeVersion = process.versions.node, run = spawnSync } = {}) {
  const problems = [];
  if (nodeVersion.split('.')[0] !== nodeMajor) {
    problems.push(`Node ${nodeMajor} is required by .nvmrc; this process uses Node ${nodeVersion}. Activate Node ${nodeMajor} in this terminal.`);
  }
  for (const [command, args, guidance] of tools) {
    const result = run(command, args, { encoding: 'utf8', timeout: 15_000, windowsHide: true });
    if (result.error?.code === 'ENOENT') problems.push(`${command} was not found on PATH. ${guidance}`);
    else if (result.error || result.status !== 0) {
      const detail = result.error?.message || result.stderr?.trim() || `exit status ${result.status}, signal ${result.signal ?? 'none'}`;
      problems.push(`${command} could not run: ${detail}. ${guidance}`);
    }
  }
  return problems;
}

export function requireDesktopToolchain() {
  const problems = desktopToolchainProblems();
  if (!problems.length) return;
  console.error(`Desktop toolchain check failed before resource preparation:\n${problems.map(problem => `- ${problem}`).join('\n')}\nSee docs/operations/local-dev.md#native-desktop.`);
  process.exit(1);
}
