// Controlled streams exercise the real bilingual producer against the pinned Rime ABI.
import {execFileSync} from 'node:child_process';
import {cachedSpawn} from './lib/windows-compile-cache.mjs';
import {copyFileSync, mkdirSync, mkdtempSync, rmSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
if (!['darwin','win32'].includes(process.platform)) throw new Error('Native macOS or Windows required.');
const windows=process.platform==='win32';
const runtime=path.join(root,'apps/desktop/resources',windows?'windows-amd64':'darwin','keyboard');
const sdk=path.join(root,'.workspace/keyboard-runtime',windows?'windows/sdk':'sdk');
const output=path.join(root,'.workspace/tests/english-translation');
mkdirSync(output,{recursive:true});
const directory=mkdtempSync(path.join(output,'run-'));
try {
  const library=windows?path.join(sdk,'lib/libArionRime.dll.a'):path.join(runtime,'lib/librime.1.dylib');
  const binary=path.join(directory,windows?'translation.exe':'translation');
  const compiler=process.env.CXX??(windows?'g++':'clang++');
  const compiled=cachedSpawn(compiler,['-std=c++17','-O2','-Wall','-Wextra','-Werror',
    '-DGLOG_USE_GLOG_EXPORT',...(windows?['-DRIME_IMPORTS','-static']:[]),'-isystem',path.join(sdk,'include'),
    ...['english_translation_test.cpp','english_lexicon.cpp'].map(file=>path.join(root,'apps/desktop/ime/shared',file)),
    library,...(windows?[]:[`-Wl,-rpath,${path.dirname(library)}`]),'-o',binary],{cwd:root,stdio:'inherit',windowsHide:true});
  if (compiled.error) throw compiled.error;
  if (compiled.status!==0) throw new Error('English translation compilation failed.');
  if (windows) copyFileSync(path.join(runtime,'lib/ArionRime.dll'),path.join(directory,'ArionRime.dll'));
  execFileSync(binary,[path.join(directory,'fixture.lexicon')],{cwd:directory,stdio:'inherit',windowsHide:true});
} finally {rmSync(directory,{recursive:true,force:true});}
