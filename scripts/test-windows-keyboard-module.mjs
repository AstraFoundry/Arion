// Isolated native module and installer-filesystem checks; no TIP registration.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
if (process.platform !== 'win32') throw new Error('Windows module checks require native Windows.');
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const source=path.join(root,'apps/desktop/ime/windows');
const common=path.join(source,'../shared');
const sdk=path.join(root,'.workspace/keyboard-runtime/windows/sdk');
const runtime=path.join(root,'apps/desktop/resources/windows-amd64/keyboard');
const output=path.join(root,'.workspace/tests/windows-keyboard-module');
mkdirSync(output,{recursive:true});
const directory=mkdtempSync(path.join(output,'run-'));
const hash=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const report={schema:1,boundary:'Production DLL load/lifecycle and synthetic installer files only; no TSF editor dispatch or machine registration',directory,sourceIdentities:{},runtimeIdentities:{},commands:[],status:'running'};
const save=()=>writeFileSync(path.join(directory,'report.json'),JSON.stringify(report,null,2)+'\n');
function run(command,args,env=process.env) {
  const started=performance.now();
  const result=spawnSync(command,args,{cwd:directory,encoding:'utf8',windowsHide:true,timeout:180000,env});
  report.commands.push({command,args,durationMs:performance.now()-started,status:result.status,error:result.error?.message,stdout:result.stdout,stderr:result.stderr}); save();
  process.stdout.write(result.stdout??''); process.stderr.write(result.stderr??'');
  if(result.error) throw result.error;
  if(result.status!==0) throw new Error(`${command} failed (${result.status})`);
}
try {
  const cpp=['learning','learning_rime','learning_windows','ranking','ranking_unicode','emoji'].map(name=>path.join(common,`${name}.cpp`));
  const files=[fileURLToPath(import.meta.url),...cpp,...['keyboard_runtime.cpp','keyboard_runtime.h','keyboard_runtime_test.cpp','control.cpp','payload_install_test.cpp','keyboard_payload.h','payload.h','json.h','identity.h','wire.h'].map(name=>path.join(source,name)),...['pinyin.c','pinyin.h','emoji.h','learning.h','learning_rime.h','ranking.h','ranking_bridge.h','ranking_unicode.h','ranking_unicode_data.h','ranking_unicode_sources.json'].map(name=>path.join(common,name))];
  for(const file of files) report.sourceIdentities[path.relative(root,file)]=hash(file);
  const library=path.join(runtime,'lib/ArionRime.dll');
  report.runtimeIdentities.library=hash(library);
  report.runtimeIdentities.importLibrary=hash(path.join(sdk,'lib/libArionRime.dll.a'));
  const cc=process.env.CC??'gcc',cxx=process.env.CXX??'g++';
  const object=path.join(directory,'pinyin.o');
  run(cc,['-std=c11','-O2','-Wall','-Wextra','-Werror','-I',path.join(sdk,'include'),'-c',path.join(common,'pinyin.c'),'-o',object]);
  const base=['-std=c++17','-DINITGUID','-O2','-Wall','-Wextra','-Werror','-static','-I',path.join(sdk,'include')];
  const module=path.join(directory,'ArionKeyboard.dll');
  run(cxx,[...base,'-DRIME_IMPORTS','-shared',path.join(source,'keyboard_runtime.cpp'),...cpp,object,path.join(sdk,'lib/libArionRime.dll.a'),'-o',module,'-lcrypt32','-lole32','-luuid','-ladvapi32']);
  copyFileSync(library,path.join(directory,'ArionRime.dll'));
  const fixture=path.join(directory,'keyboard-runtime-test.exe');
  run(cxx,[...base,'-municode',path.join(source,'keyboard_runtime_test.cpp'),object,'-o',fixture]);
  const installer=path.join(directory,'payload-install-test.exe');
  run(cxx,[...base,'-municode',path.join(source,'payload_install_test.cpp'),'-o',installer,'-lole32','-loleaut32','-luuid','-ladvapi32','-lshell32','-lgdi32']);
  const env={SystemRoot:process.env.SystemRoot,WINDIR:process.env.SystemRoot,PATH:path.join(process.env.SystemRoot,'System32'),TEMP:process.env.TEMP,TMP:process.env.TMP,APPDATA:process.env.APPDATA};
  run(fixture,[module,path.join(runtime,'data'),path.join(directory,'profile')],env);
  run(installer,[],env);
  report.changedSources=Object.entries(report.sourceIdentities).filter(([name,digest])=>hash(path.join(root,name))!==digest).map(([name])=>name);
  if(report.changedSources.length) throw new Error(`Sources changed during compilation/test; rerun after edits settle: ${report.changedSources.join(', ')}`);
  report.binaryIdentities=Object.fromEntries([module,fixture,installer].map(file=>[path.basename(file),hash(file)]));
  report.status='passed';
} catch(error) { report.status='failed'; report.error=error.message; throw error; }
finally { save(); console.log(`Module evidence: ${path.join(directory,'report.json')}`); }
