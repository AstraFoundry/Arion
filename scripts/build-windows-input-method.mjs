import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyWindowsPEDependencies } from './lib/windows-pe.mjs';
import { recordWindowsKeyboardModule, verifyWindowsKeyboardRuntime } from './lib/windows-keyboard.mjs';

if (process.platform !== 'win32') process.exit(0);
if (process.arch !== 'x64') throw new Error('The Windows TSF text service requires Windows x64');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'apps/desktop/ime/windows');
const output = path.join(root, 'apps/desktop/resources/windows-amd64/ime');
const objects = path.join(root, '.workspace/windows-tsf/build');
mkdirSync(output, { recursive: true });
mkdirSync(objects, { recursive: true });
const compiler = process.env.CXX ?? 'g++';
const cCompiler = process.env.CC ?? 'gcc';
const run = (command,args) => execFileSync(command,args,{cwd:root,stdio:'inherit',windowsHide:true});
if (/(?:^|[/\\])cl(?:\.exe)?$/i.test(compiler)) throw new Error('The Windows encrypted keyboard module requires the same MinGW-w64 GCC ABI as the pinned Rime runtime.');
run(process.execPath,[path.join(root,'scripts/stage-keyboard-runtime.mjs')]);
const keyboard = path.join(root,'apps/desktop/resources/windows-amd64/keyboard');
const sdk = path.join(root,'.workspace/keyboard-runtime/windows/sdk');
const common = path.join(source,'../shared');
const cObjects = [];
for (const name of ['pinyin','input','menu']) {
  const object=path.join(objects,`${name}.o`);
  run(cCompiler,['-std=c11','-O2','-Wall','-Wextra','-Werror','-I',path.join(sdk,'include'),'-c',path.join(common,`${name}.c`),'-o',object]);
  cObjects.push(object);
}
const base=['-std=c++17','-DINITGUID','-O2','-Wall','-Wextra','-Werror','-static','-I',path.join(sdk,'include')];
run(compiler,[...base,'-DRIME_IMPORTS','-shared',path.join(source,'keyboard_runtime.cpp'),...['learning','learning_rime','learning_windows','ranking','ranking_unicode'].map(name=>path.join(common,`${name}.cpp`)),cObjects[0],path.join(sdk,'lib/libArionRime.dll.a'),'-o',path.join(keyboard,'lib/ArionKeyboard.dll'),'-lcrypt32','-lole32','-luuid','-ladvapi32']);
recordWindowsKeyboardModule(keyboard);
verifyWindowsKeyboardRuntime(keyboard);
const keyboardManifestSHA256=createHash('sha256').update(readFileSync(path.join(keyboard,'runtime.json'))).digest('hex');
// The COM entry point must remain byte-identical across core-only updates.
// In particular, do not bind it to the changing keyboard manifest or PE time.
const loader = path.join(output, 'ArionTSFLoader.dll');
run(compiler,[...base,'-Wl,--no-insert-timestamp','-Wl,--image-base,0x180000000','-shared',path.join(source,'loader.cpp'),'-o',loader,'-lole32','-luuid','-ladvapi32','-lshell32']);
const loaderSHA256=createHash('sha256').update(readFileSync(loader)).digest('hex');
for (const [input, filename, shared] of [['service.cpp', 'ArionTSF.dll', true], ['control.cpp', 'arion-ime.exe', false]]) {
  const destination = path.join(output, filename);
  const args = [...base,`-DARION_KEYBOARD_MANIFEST_SHA256="${keyboardManifestSHA256}"`,...(!shared?[`-DARION_TSF_LOADER_SHA256="${loaderSHA256}"`]:[]),...(shared ? ['-shared'] : ['-municode']),path.join(source,input),...(shared?cObjects:[]),'-o',destination,'-lole32','-loleaut32','-luuid','-ladvapi32','-lshell32','-lgdi32'];
  run(compiler,args);
}
verifyWindowsPEDependencies(['ArionTSFLoader.dll', 'ArionTSF.dll', 'arion-ime.exe'].map(name => path.join(output, name)));
const files = Object.fromEntries(['ArionTSFLoader.dll', 'ArionTSF.dll', 'arion-ime.exe'].map(name => [name, createHash('sha256').update(readFileSync(path.join(output, name))).digest('hex')]));
writeFileSync(path.join(output, 'runtime.json'), JSON.stringify({ schema: 2, architecture: 'amd64', keyboardManifestSHA256, files }, null, 2) + '\n');
console.log('Built Windows x64 TSF host, encrypted keyboard module and complete runtime closure.');
