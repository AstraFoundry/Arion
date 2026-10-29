import { execFileSync } from 'node:child_process';

// Resolve a persistent identity before staging expensive native resources. Ad hoc
// signatures change the app's designated requirement on every edited build.
export function developmentSigningIdentity(env, run = execFileSync) {
  const requested = env.MACOS_DEV_SIGNING_IDENTITY?.trim();
  if (!requested || requested === '-') {
    throw new Error('Set MACOS_DEV_SIGNING_IDENTITY to a persistent code-signing certificate in your Keychain. See docs/operations/local-dev.md. Ad hoc development signing cannot preserve permissions across rebuilds.');
  }
  const output = run('security', ['find-identity', '-v', '-p', 'codesigning'], { encoding: 'utf8' });
  const identities = [...output.matchAll(/^\s*\d+\)\s+([A-Fa-f0-9]{40})\s+"([^"]+)"/gm)];
  const matches = identities.filter(([, hash, name]) => hash.toUpperCase() === requested.toUpperCase() || name === requested);
  if (matches.length !== 1) throw new Error('MACOS_DEV_SIGNING_IDENTITY must match exactly one valid installed identity (name or SHA-1). Run security find-identity -v -p codesigning.');
  return matches[0][1].toUpperCase();
}

export function desktopProcesses(output) {
  return output.split('\n').flatMap(line => {
    const match = /^\s*(\d+)\s+(.+?)\s*$/.exec(line);
    if (!match) return [];
    const [, pid, executable] = match;
    const development = executable.endsWith('/Arion Dev.app/Contents/MacOS/Arion Dev');
    const release = executable.endsWith('/Arion.app/Contents/MacOS/Arion');
    return development || release ? [{ pid: Number(pid), executable, development }] : [];
  });
}

export function requireDesktopStopped(processes) {
  if (processes.length) throw new Error(`Quit the running Arion from its menu before switching or starting a development session:\n${processes.map(p => `${p.pid}: ${p.executable}`).join('\n')}\nNo process was terminated.`);
}

export function macOSDesktopProcesses() {
  return desktopProcesses(execFileSync('ps', ['-axo', 'pid=,comm='], { encoding: 'utf8' }));
}
