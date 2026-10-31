import { buildCandidateFonts } from './lib/windows-candidate-fonts.mjs';
// Owned ACP keyboard checks in the current desktop token, including elevated hosts.
import { spawn } from 'node:child_process';
import { cachedSpawn } from './lib/windows-compile-cache.mjs';
import { createHash } from 'node:crypto';
import { cpSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import net from 'node:net';
import { verifyWindowsKeyboardRuntime } from './lib/windows-keyboard.mjs';

if (process.platform !== 'win32') throw new Error('Native Windows is required.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const args = process.argv.slice(2);
if (args.length && (args.length !== 2 || args[0] !== '--runtime')) throw new Error('Usage: node scripts/test-windows-keyboard-host.mjs [--runtime KEYBOARD_DIRECTORY]');
const runtime = args.length ? path.resolve(args[1]) : path.join(root, 'apps/desktop/resources/windows-amd64/keyboard');
verifyWindowsKeyboardRuntime(runtime);
const output = path.join(root, '.workspace/tests/windows-keyboard-host');
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'run-'));
const source = path.join(root, 'apps/desktop/ime/windows');
const common = path.join(source, '../shared');
const keyboard = path.join(directory, 'keyboard');
cpSync(runtime, keyboard, { recursive: true });
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const report = { schema: 1, directory, status: 'running', boundary: 'Real token admission and manually dispatched production TSF keys in owned ACP stores with an isolated encrypted profile. No installed registration, system key dispatch or user-document changes.', commands: [], cases: [], sources: {} };
const sources = new Set();
function collect(file) {
  file = path.resolve(file);
  if (sources.has(file)) return;
  sources.add(file);
  for (const match of readFileSync(file, 'utf8').matchAll(/^\s*#\s*include\s*"([^"]+)"/gm)) collect(path.resolve(path.dirname(file), match[1]));
}
for (const name of ['service.cpp', 'textstore_fixture.cpp', 'host_security_test.cpp', 'keyboard_options_test.cpp']) collect(path.join(source, name));
for (const name of ['pinyin.c', 'input.c', 'menu.c']) collect(path.join(common, name));
sources.add(fileURLToPath(import.meta.url));
for (const file of sources) report.sources[path.relative(root, file)] = hash(file);
const save = () => writeFileSync(path.join(directory, 'report.json'), JSON.stringify(report, null, 2) + '\n');
function run(command, argv) {
  const result = cachedSpawn(command, argv, { cwd: directory, windowsHide: true, encoding: 'utf8', timeout: 120000 });
  report.commands.push({ command, args: argv, status: result.status, error: result.error?.message, stdout: result.stdout, stderr: result.stderr });
  save();
  if (result.error || result.status !== 0) throw result.error ?? new Error(result.stderr || command + ' failed: ' + result.status);
  return result.stdout;
}
let fixture, server, acceptedConnections = 0;
const connections = new Set();
try {
  const cxx = process.env.CXX ?? 'g++', cc = process.env.CC ?? 'gcc';
  report.compiler = run(cxx, ['--version']).split('\n')[0];
  const base = ['-std=c++17', '-DINITGUID', '-O2', '-Wall', '-Wextra', '-Werror', '-static', '-isystem', path.join(keyboard, 'include')];
  const policy = path.join(directory, 'host-security-test.exe');
  run(cxx, [...base, path.join(source, 'host_security_test.cpp'), '-o', policy, '-ladvapi32']);
  report.token = JSON.parse(run(policy, []));
  const optionsTest = path.join(directory, 'keyboard-options-test.exe');
  run(cxx, [...base, '-municode', path.join(source, 'keyboard_options_test.cpp'), '-o', optionsTest, '-lole32', '-ladvapi32', '-luuid']);
  run(optionsTest, [path.join(directory, 'options-fixture')]);
  report.cases.push('atomic mode options, concurrent writers, unsafe ACL and reparse refusal');
  const objects = ['pinyin', 'input', 'menu'].map(name => {
    const object = path.join(directory, name + '.o');
    run(cc, ['-std=c11', '-O2', '-Wall', '-Wextra', '-Werror', '-I', path.join(keyboard, 'include'), '-c', path.join(common, name + '.c'), '-o', object]);
    return object;
  });
  objects.push(buildCandidateFonts(root, cxx, directory, run));
  const dll = path.join(directory, 'ArionTSF.dll'), editor = path.join(directory, 'textstore-fixture.exe');
  const manifest = path.join(keyboard, 'runtime.json');
  run(cxx, [...base, '-DARION_TSF_TESTING', '-DARION_TSF_MANUAL_KEYS', '-DARION_KEYBOARD_MANIFEST_SHA256="' + hash(manifest) + '"', '-shared', path.join(source, 'service.cpp'), ...objects, '-o', dll, '-lole32', '-loleaut32', '-luuid', '-ladvapi32', '-lshell32', '-lgdi32', '-ldwrite', '-ld2d1']);
  const production = path.join(directory, 'ArionTSF-production.dll');
  run(cxx, [...base, '-DARION_KEYBOARD_MANIFEST_SHA256="' + hash(manifest) + '"', '-shared', path.join(source, 'service.cpp'), ...objects, '-o', production, '-lole32', '-loleaut32', '-luuid', '-ladvapi32', '-lshell32', '-lgdi32', '-ldwrite', '-ld2d1']);
  run(cxx, [...base, '-DARION_TSF_MANUAL_KEYS', '-municode', path.join(source, 'textstore_fixture.cpp'), '-o', editor, '-lole32', '-loleaut32', '-luuid', '-lgdi32', '-ladvapi32']);
  report.binaries = Object.fromEntries([policy, optionsTest, dll, production, editor, manifest, path.join(keyboard, 'lib/ArionKeyboard.dll'), path.join(keyboard, 'lib/ArionRime.dll')].map(file => [path.relative(directory, file), hash(file)]));
  const pipe = "\\\\.\\pipe\\Arion.TSF.Test.KeyboardHost." + process.pid;
  server = net.createServer(socket => { ++acceptedConnections; connections.add(socket); socket.on('error', () => {}); socket.on('close', () => connections.delete(socket)); });
  await new Promise((resolve, reject) => { server.once('error', reject); server.listen(pipe, resolve); });
  fixture = spawn(editor, [dll], { cwd: directory, windowsHide: true, env: { ...process.env, ARION_TSF_TEST_USER_DIRECTORY: path.join(directory, 'profile'), ARION_TSF_TEST_PIPE: pipe } });
  let buffer = '', diagnostics = '', closed = false;
  const lines = [], waiting = [];
  fixture.stdout.on('data', data => {
    buffer += data;
    let end;
    while ((end = buffer.indexOf('\n')) >= 0) {
      const line = buffer.slice(0, end).trim(); buffer = buffer.slice(end + 1);
      if (waiting.length) waiting.shift().resolve(JSON.parse(line)); else lines.push(JSON.parse(line));
    }
  });
  fixture.stderr.on('data', data => { diagnostics += data; report.diagnostics = diagnostics; });
  const exit = new Promise((resolve, reject) => {
    fixture.once('error', reject);
    fixture.once('exit', code => { closed = true; for (const item of waiting.splice(0)) item.reject(new Error('Fixture exited ' + code + ': ' + diagnostics)); resolve(code); });
  });
  async function next() {
    if (lines.length) return lines.shift();
    if (closed) throw new Error('Fixture closed: ' + diagnostics);
    let item, timer;
    try {
      return await Promise.race([
        new Promise((resolve, reject) => { item = { resolve, reject }; waiting.push(item); }),
        new Promise((_, reject) => { timer = setTimeout(() => reject(new Error('Fixture response timeout: ' + diagnostics)), 10000); })
      ]);
    } finally { clearTimeout(timer); const index = waiting.indexOf(item); if (index >= 0) waiting.splice(index, 1); }
  }
  async function command(text) { fixture.stdin.write(text + '\n'); const result = await next(); (report.fixture ??= []).push({ command: text, result }); return result; }
  // TSF can defer a granted edit; wait for its observable result, never resend keys.
  async function observe(text, expected) {
    const deadline = Date.now() + 3000;
    let result;
    do {
      result = await command(text);
      if (expected(result)) return result;
      await new Promise(resolve => setTimeout(resolve, 20));
    } while (Date.now() < deadline);
    throw new Error('TSF observation did not settle: ' + text + ' ' + JSON.stringify(result));
  }
  async function key(code, modifiers = 0, up = false) { return command('key:' + code + ':' + modifiers + (up ? ':up' : '')); }
  assert.equal((await next()).ready, true);
  await command('reset');
  let prefix = '';
  for (const ch of 'NIHAO') {
    assert.equal((await key(ch.charCodeAt(0))).eaten, true);
    prefix += ch.toLowerCase();
    await observe('text', value => value.text.replace(/[ '\u2019]/g, '') === 'A🙂' + prefix + 'Z');
  }
  const menu = await observe('menu', value => value.strings.includes('你好'));
  assert.ok(menu.strings.includes('你好'));
  assert.equal((await key(32)).eaten, true);
  assert.equal((await observe('text', value => value.text === 'A🙂你好Z')).text, 'A🙂你好Z');
  report.cases.push('real Pinyin candidates and confirmed Chinese commit');
  await command('reset'); await command('writelayout');
  await key(78); await observe('text', value => value.text === 'A🙂nZ'); await observe('menu', value => value.visible);
  const anchor = await command('menu');
  const surfacesBefore = await command('surfaceevents');
  const layoutBefore = await command('layoutstate');
  prefix = 'n';
  for (const ch of 'IHAO') {
    await key(ch.charCodeAt(0));
    prefix += ch.toLowerCase();
    await observe('text', value => value.text.replace(/[ '\u2019]/g, '') === 'A🙂' + prefix + 'Z');
    const position = await observe('menu', value => value.visible);
    assert.deepEqual([position.x, position.y], [anchor.x, anchor.y], 'Candidate position stays at the conversion start');
  }
  await key(8); await observe('text', value => value.text.replace(/[ '\u2019]/g, '') === 'A🙂nihaZ');
  const edited = await observe('menu', value => value.visible);
  assert.deepEqual([edited.x, edited.y], [anchor.x, anchor.y]);
  await key(79); await observe('text', value => value.text.replace(/[ '\u2019]/g, '') === 'A🙂nihaoZ');
  for (const code of [37, 36, 35, 34, 33]) {
    await key(code);
    const position = await observe('menu', value => value.visible);
    assert.deepEqual([position.x, position.y], [anchor.x, anchor.y]);
  }
  const surfacesAfter = await command('surfaceevents');
  assert.equal(surfacesAfter.hides, surfacesBefore.hides, 'Continuous typing must not hide and reshow the candidate window during its write session');
  assert.equal((await command('layoutstate')).writeExtents, layoutBefore.writeExtents, 'Candidate geometry is queried only after the write session');
  await key(27); await observe('menu', value => !value.visible); await command('anylayout');
  await command('reset'); await command('scroll'); await key(78); await observe('text', value => value.text === 'A🙂nZ');
  const newAnchor = await observe('menu', value => value.visible);
  assert.equal(newAnchor.x, anchor.x + 40, 'A new conversion acquires a fresh anchor');
  await command('clipped'); await observe('menu', value => !value.visible);
  await command('unclipped'); await observe('menu', value => value.visible);
  await command('delaylayout'); await key(73); await observe('text', value => value.text === 'A🙂niZ');
  await observe('menu', value => !value.visible);
  await command('readlocks'); await command('layout'); await observe('menu', value => value.visible);
  await command('writelocks'); await command('instantlayout');
  await key(27); await observe('menu', value => !value.visible);
  await command('reset'); prefix = '';
  for (const ch of 'NIXIAOLIAN') {
    await key(ch.charCodeAt(0)); prefix += ch.toLowerCase();
    await observe('text', value => value.text.replace(/[ '\u2019]/g, '') === 'A🙂' + prefix + 'Z');
  }
  const partial = await observe('menu', value => value.visible && value.strings.includes('你'));
  const selectedPrefix = partial.strings.indexOf('你'); assert.ok(selectedPrefix < 9);
  await key(49 + selectedPrefix);
  await observe('text', value => value.text.replace(/[ '\u2019]/g, '') === 'A🙂你xiaolianZ');
  const remainder = await observe('menu', value => value.visible && value.strings.includes('笑脸'));
  assert.deepEqual([remainder.x, remainder.y], [partial.x, partial.y], 'Partial selection preserves the original conversion anchor');
  await key(27); await observe('menu', value => !value.visible);
  report.cases.push('partial selection preserves the anchor while the suffix remains composing');
  report.cases.push('conversion-start anchor persists through typing, caret movement and paging; new input reacquires it; clipping and missing layout still hide safely');
  report.cases.push('continuous typing defers geometry to read sessions without candidate hide/show transitions');
  const chinese = (await command('mode')).mode;
  await key(160, 1); await key(160, 0, true);
  const english = (await command('mode')).mode;
  assert.notEqual(english, chinese);
  await key(32, 1);
  assert.equal((await command('mode')).mode, chinese);
  assert.equal((await command('action:2')).hr, 0);
  assert.equal((await command('mode')).mode, english);
  await command('action:2');
  report.cases.push('standalone Shift, Shift+Space and language-menu switching');
  assert.equal((await command('action:3')).hr, 0);
  assert.equal((await command('action:4')).hr, 0);
  assert.deepEqual(JSON.parse(readFileSync(path.join(directory, 'profile/options.json'), 'utf8')), { fullWidth: true, englishPunctuation: true });
  report.cases.push('elevated-created private mode options save with current-user ownership');
  await command('password');
  assert.equal((await key(78)).eaten, false);
  await command('edit'); await command('readonly');
  assert.equal((await key(78)).eaten, false);
  await command('writable');
  report.cases.push('password and read-only fields refuse typing');
  if (report.token.elevated) {
    assert.equal((await command('action:1')).hr, 1);
    assert.equal(acceptedConnections, 0);
    report.cases.push('elevated keyboard host keeps voice menu disabled and never connects to pipe');
  }
  fixture.stdin.write('quit\n');
  assert.equal(await exit, 0);
  report.diagnostics = diagnostics;
  for (const [file, expected] of Object.entries(report.sources)) assert.equal(hash(path.join(root, file)), expected, 'Source changed during check: ' + file);
  report.status = 'passed';
} catch (error) { report.status = 'failed'; report.error = error.stack; process.exitCode = 1; }
finally {
  if (fixture && fixture.exitCode === null) fixture.kill();
  for (const socket of connections) socket.destroy();
  if (server) await new Promise(resolve => server.close(resolve));
  save();
  console.log('Windows keyboard host evidence: ' + path.join(directory, 'report.json'));
}
if (report.status !== 'passed') throw new Error(report.error);
