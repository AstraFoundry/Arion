// Real DLL loading and callback-only registration contracts. No OS registration.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
if (process.platform !== 'win32' || process.arch !== 'x64') throw new Error('Native Windows x64 is required.');
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const source=path.join(root,'apps/desktop/ime/windows');
const output=path.join(root,'.workspace/tests/windows-tsf-loader');
mkdirSync(output,{recursive:true});
const directory=mkdtempSync(path.join(output,'run-'));
const sha=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const report={schema:1,boundary:'Owned fake native COM cores, production loader path refusal, deterministic binary identity and callback-only registration/rollback; no install, registry writes, profile changes or editor input',directory,sources:{},commands:[],status:'running'};
const save=()=>writeFileSync(path.join(directory,'report.json'),JSON.stringify(report,null,2)+'\n');
function run(phase,command,args) {
  const result=spawnSync(command,args,{cwd:directory,encoding:'utf8',windowsHide:true,timeout:120000,maxBuffer:8*1024*1024});
  report.commands.push({phase,command,args,status:result.status,error:result.error?.message,stdout:result.stdout,stderr:result.stderr});save();
  process.stdout.write(result.stdout??'');process.stderr.write(result.stderr??'');
  if(result.error) throw result.error;
  if(result.status!==0) throw new Error(`${phase} failed (${result.status})`);
  return result;
}
try {
  const files=['loader.cpp','loader.h','loader_fixture.cpp','loader_test.cpp','control.cpp','upgrade_test.cpp','payload_install_test.cpp','identity.h','payload.h','keyboard_payload.h','wire.h','json.h'];
  for(const file of files) report.sources[path.relative(root,path.join(source,file))]=sha(path.join(source,file));
  report.sources[path.relative(root,fileURLToPath(import.meta.url))]=sha(fileURLToPath(import.meta.url));
  const compiler=process.env.CXX??'g++';
  report.compiler=run('compiler',compiler,['--version']).stdout.split('\n')[0].trim();
  const flags=['-std=c++17','-O2','-Wall','-Wextra','-Werror','-static'];
  const links=['-lole32','-loleaut32','-luuid','-ladvapi32','-lshell32'];
  const dll=(name,extra=[])=>{
    const target=path.join(directory,name);
    run(`compile-${name}`,compiler,[...flags,'-shared','-Wl,--no-insert-timestamp','-Wl,--image-base,0x180000000',...extra,path.join(source,'loader.cpp'),'-o',target,...links]);
    return target;
  };
  const production=dll('ArionTSFLoader.dll');
  mkdirSync(path.join(directory,'repeat'));
  const repeated=dll('repeat/ArionTSFLoader.dll');
  if(sha(production)!==sha(repeated)) throw new Error('Loader identity changed across identical source builds.');
  const test=dll('ArionTSFLoader-test.dll',['-DARION_TSF_LOADER_TEST']);
  const cores=[];
  for(const number of [1,2]) {
    const target=path.join(directory,`core-${number}.dll`);
    run(`compile-core-${number}`,compiler,[...flags,'-shared','-Wl,--no-insert-timestamp',`-DARION_LOADER_FIXTURE_VERSION=${number}`,path.join(source,'loader_fixture.cpp'),'-o',target,...links]);
    cores.push(target);
  }
  const fixture=path.join(directory,'loader-test.exe');
  run('compile-loader-test',compiler,[...flags,'-municode',path.join(source,'loader_test.cpp'),'-o',fixture,...links]);
  run('native-loader-lifecycle',fixture,[test,...cores,production]);
  for(const name of ['upgrade','payload_install']) {
    const target=path.join(directory,`${name}-test.exe`);
    run(`compile-${name}`,compiler,[...flags,'-municode','-DINITGUID',`-DARION_TSF_LOADER_SHA256="${sha(production)}"`,path.join(source,`${name}_test.cpp`),'-o',target,...links]);
    run(`callback-filesystem-${name}`,target,name==='payload_install'?[test]:[]);
  }
  report.binaryIdentities=Object.fromEntries([production,repeated,test,...cores,fixture].map(file=>[path.relative(directory,file),sha(file)]));
  report.changedSources=Object.entries(report.sources).filter(([file,digest])=>sha(path.join(root,file))!==digest).map(([file])=>file);
  if(report.changedSources.length) throw new Error(`Sources changed during gate: ${report.changedSources.join(', ')}`);
  report.status='passed';
} catch(error) {report.status='failed';report.error=error.message;process.exitCode=1;}
finally {save();console.log(`Windows TSF loader evidence: ${path.join(directory,'report.json')}`);}
