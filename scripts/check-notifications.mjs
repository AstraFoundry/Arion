#!/usr/bin/env node
// Native notification evidence uses an isolated sender and test-owned files.
import { execFile, spawn } from 'node:child_process';
import { promisify } from 'node:util';
import { appendFile, mkdir, readFile, readdir, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { randomUUID } from 'node:crypto';

const run = promisify(execFile);
const repo = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const module = join(repo, 'third_party/mygo');
const args = new Set(process.argv.slice(2));
const native = args.has('--native') || args.has('--prepare') || args.has('--interactive');
const platform = process.platform === 'darwin' ? 'darwin' : process.platform === 'win32' ? 'windows' : process.platform;
const arch = process.arch === 'x64' ? 'amd64' : process.arch;
const env = { ...process.env, GOWORK: 'off', CGO_ENABLED: '0' };

async function command(cmd, argv, options = {}) {
  const result = await run(cmd, argv, { cwd: module, env, timeout: 180_000, maxBuffer: 8 << 20, ...options });
  if (result.stdout) process.stdout.write(result.stdout);
  if (result.stderr) process.stderr.write(result.stderr);
  return result;
}
async function waitFor(fn, label, timeout = 30_000) {
  const end = Date.now() + timeout;
  while (Date.now() < end) {
    const result = await fn();
    if (result) return result;
    await new Promise(resolve => setTimeout(resolve, 100));
  }
  throw new Error(`Timed out waiting for ${label}`);
}

if (!native) {
  await command('go', ['test', './internal/notification', '-count=1']);
  await command('go', ['test', '.', './cmd/mygo', '-run', 'Notification|TestBeforeRun|TestConfigTypes', '-count=1']);
  for (const target of ['amd64', 'arm64']) await command('go', ['build', './...'], { env: { ...env, GOOS: 'windows', GOARCH: target } });
  console.log('Native notification unit checks and Windows cross-builds passed. Runtime acceptance requires --native or --interactive on the target OS.');
} else {
  if (!['darwin', 'windows'].includes(platform)) throw new Error('Native notification acceptance requires macOS or Windows.');
  const outArg = process.argv.indexOf('--out');
  const out = outArg >= 0 ? resolve(process.argv[outArg + 1]) : join(repo, '.workspace/e2e/notifications', `${platform}-${arch}-${randomUUID()}`);
  await mkdir(out, { recursive: true });
  await command('go', ['run', './cmd/mygo', 'build', '-platform', `${platform}/${arch}`, '-skip-dmg', '-o', join(out, 'build'), './examples/notification-probe']);
  const built = join(out, 'build', `${platform}-${arch}`);
  const bundle = join(built, 'Arion Notification Probe.app');
  const exe = platform === 'darwin' ? join(bundle, 'Contents/MacOS/Arion Notification Probe') : join(built, 'Arion Notification Probe.exe');
  const report = join(out, 'events.jsonl');
  const control = join(out, 'commands.jsonl');
  await writeFile(report, ''); await writeFile(control, '');
  await writeFile(join(dirname(exe), 'notification-probe.json'), JSON.stringify({ report, control }));
  // The sidecar is test-owned. It must be present before a cold COM launch.
  if (platform === 'darwin') await command('/usr/bin/codesign', ['--force', '--deep', '--sign', '-', bundle]);
  if (platform === 'windows') await command(exe, ['--mygo-register-notifications']);
  const events = async () => (await readFile(report, 'utf8')).trim().split('\n').filter(Boolean).map(x => JSON.parse(x));
  const send = async (op, id = '') => {
    const token = randomUUID();
    await appendFile(control, JSON.stringify({ token, op, id }) + '\n');
    return token;
  };
  const ack = async (token, timeout = 30_000) => waitFor(async () => (await events()).find(x => x.event === 'command' && x.token === token), token, timeout);
  const launch = async () => {
    const previous = new Set((await events()).filter(x => x.event === 'launch').map(x => x.pid));
    if (platform === 'darwin') await command('/usr/bin/open', ['-n', bundle]);
    else { const child = spawn(exe, [], { detached: true, stdio: 'ignore', env }); child.unref(); }
    return waitFor(async () => (await events()).find(x => x.event === 'ready' && !previous.has(x.pid)), 'native ready');
  };
  await launch();
  const permission = await ack(await send('query'));
  console.log(JSON.stringify({ out, bundle: platform === 'darwin' ? bundle : undefined, exe, report, control, permission }, null, 2));
  if (args.has('--prepare')) {
    console.log('Owned probe is running. Append {token,op,id} JSONL commands. request explicitly opens its authorization prompt; show delivers, quit preserves history, clear removes only this probe sender.');
  } else {
    let registered = platform === 'windows';
    let completed = false;
    try {
      if (permission.error || !['unknown', 'denied', 'allowed'].includes(permission.status)) {
        throw new Error(`Native notification permission query failed: ${JSON.stringify(permission)}`);
      }
      if (args.has('--interactive')) {
        let state = permission;
        if (state.status === 'unknown') {
          console.log('Awaiting explicit Allow/Deny for the isolated Arion Notification Probe sender.');
          state = await ack(await send('request'), 120_000);
        }
        if (state.error || state.status !== 'allowed') throw new Error(`Notification presentation unavailable: ${JSON.stringify(state)}`);
        for (const [name, action, cold] of [['warm-view', 'view', false], ['warm-later', 'later', false], ['cold-body', '', true], ['cold-later', 'later', true]]) {
          const id = `probe/stable/1.2.3/${name}`;
          const shown = await ack(await send('show', id));
          if (shown.error) throw new Error(shown.error);
          if (cold) {
            const quit = await ack(await send('quit'));
            await waitFor(async () => (await events()).find(x => x.event === 'exit' && x.pid === quit.pid), 'probe quit');
          }
          console.log(`Awaiting ${name}: ${action || 'notification body'} activation from the actual OS notification.`);
          const activation = await waitFor(async () => (await events()).find(x => x.event === 'activation' && x.id === id && x.action === action), name, 120_000);
          if (!activation.ready || cold === (activation.pid === shown.pid)) throw new Error(`Incorrect warm/cold ownership: ${JSON.stringify(activation)}`);
          await ack(await send('remove', id));
        }
      }
      await ack(await send('clear'));
      const quit = await ack(await send('quit'));
      await waitFor(async () => (await events()).find(x => x.event === 'exit' && x.pid === quit.pid), 'probe exit');
      completed = true;
      console.log('Native permission/query lifecycle passed; banner/action/cold-launch acceptance is recorded only with --interactive.');
    } finally {
      const cleanupErrors = [];
      try {
        const observed = await events();
        const lastLaunch = observed.filter(x => x.event === 'launch').at(-1);
        const exited = lastLaunch && observed.some(x => x.event === 'exit' && x.pid === lastLaunch.pid);
        if (!exited) {
          await ack(await send('clear'));
          const quit = await ack(await send('quit'));
          await waitFor(async () => (await events()).find(x => x.event === 'exit' && x.pid === quit.pid), 'cleanup exit');
        } else if (!completed && args.has('--interactive') && observed.some(x => x.event === 'command' && x.op === 'show' && !x.error)) {
          // A timed-out cold click may leave history while the process is quit.
          await launch();
          await ack(await send('clear'));
          const quit = await ack(await send('quit'));
          await waitFor(async () => (await events()).find(x => x.event === 'exit' && x.pid === quit.pid), 'cold cleanup exit');
        }
      } catch (error) { cleanupErrors.push(String(error)); }
      // Remove only the owned probe sender; never touch Arion production identity.
      if (registered) {
        try { await command(exe, ['--mygo-unregister-notifications']); registered = false; }
        catch (error) { cleanupErrors.push(String(error)); }
      }
      await writeFile(join(out, 'manifest.json'), JSON.stringify({ platform, arch, sender: 'si.astrafoundry.arion.notification-probe', interactive: args.has('--interactive'), cleanupErrors, files: await readdir(out) }, null, 2));
      if (cleanupErrors.length) throw new Error(`Owned probe cleanup failed: ${cleanupErrors.join('; ')}`);
    }
  }
}
