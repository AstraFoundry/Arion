// Explicit, isolated Developer ID process proof; never installs an input source.
import assert from 'node:assert/strict';
import {spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {copyFileSync,mkdirSync,mkdtempSync,readFileSync,readdirSync,writeFileSync} from 'node:fs';
import path from 'node:path';
import {buildIMETypingFixture} from './lib/build-ime-typing-fixture.mjs';
import {fileURLToPath} from 'node:url';
if(process.platform!=='darwin')throw new Error('Publisher process validation requires macOS.');
const vocabulary=process.argv.length===3&&process.argv[2]==='--vocabulary';
if(process.argv.length!==2&&!vocabulary)throw new Error('Usage: node scripts/test-ime-publisher.mjs [--vocabulary]');
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const parent=path.join(root,'.workspace/tests');mkdirSync(parent,{recursive:true});
const work=mkdtempSync(path.join(parent,'ime-publisher-')),evidence=path.join(work,'evidence');mkdirSync(evidence);
const save=(name,value)=>writeFileSync(path.join(evidence,name),JSON.stringify(value,null,2)+'\n');
const hash=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const commands=[];
const run=(command,args,name,options={})=>{
 const result=spawnSync(command,args,{cwd:root,encoding:'utf8',timeout:300000,maxBuffer:16*1024*1024,...options});
 writeFileSync(path.join(evidence,`${name}.stdout`),result.stdout??'');writeFileSync(path.join(evidence,`${name}.stderr`),result.stderr??'');
 commands.push({command,args,status:result.status,signal:result.signal,error:result.error?.message??null});save('commands.json',commands);
 if(result.error||result.status!==0)throw new Error(`${name} failed; retained evidence: ${evidence}`);
 return result.stdout;
};
save('status.json',{status:'preparing'});
try {
 // Read only certificate identities. No private keys, credentials or profiles
 // are exported; codesign performs the authorized local signing operation.
 const identities=spawnSync('security',['find-identity','-v','-p','codesigning'],{encoding:'utf8'});
 if(identities.status!==0)throw new Error('Cannot inspect available code-signing identities.');
 const available=[...(identities.stdout??'').matchAll(/^\s*\d+\)\s+([A-Fa-f0-9]{40})\s+"(Developer ID Application: [^"]+)"/gm)];
 const selected=process.env.MACOS_SIGNING_IDENTITY?available.filter(m=>m[2]===process.env.MACOS_SIGNING_IDENTITY):available;
 if(selected.length!==1)throw new Error('Select one available Developer ID Application identity with MACOS_SIGNING_IDENTITY.');
 const certificate=selected[0][1],config=JSON.parse(readFileSync(path.join(root,'apps/desktop/desktop.json')));
 const plist=JSON.parse(run('plutil',['-convert','json','-o','-',path.join(root,'apps/desktop/ime/macos/Info.plist')],'input-metadata'));
 const sources=['scripts/test-ime-publisher.mjs',...['peer_signed_darwin_test.go','peer_darwin_test.go','peer_darwin.go','peer_darwin.h','identity.go'].map(name=>`apps/desktop/infrastructure/imebridge/${name}`)];
 if(vocabulary)sources.push('apps/desktop/infrastructure/imebridge/peer_native_darwin_test.go','apps/desktop/ime/macos/tests/vocabulary_peer.mm','apps/desktop/ime/macos/tests/typing_fixture.h','scripts/lib/build-ime-typing-fixture.mjs');
 mkdirSync(path.join(evidence,'sources'));
 if(vocabulary){
  sources.push('scripts/lib/build-ime-learning.mjs','apps/desktop/resources/darwin/keyboard/runtime.json');
  for(const dir of ['apps/desktop/ime/macos','apps/desktop/ime/shared','apps/desktop/infrastructure/imebridge','packages/core/domain'])
   for(const name of readdirSync(path.join(root,dir)))if(/\.(h|m|mm|c|cpp|swift|go)$/.test(name))sources.push(`${dir}/${name}`);
 }
 for(const file of new Set(sources)){
  const destination=path.join(evidence,'sources',file);mkdirSync(path.dirname(destination),{recursive:true});copyFileSync(path.join(root,file),destination);
 }
 save('inputs.json',{sources:Object.fromEntries(sources.map(file=>[file,hash(path.join(root,file))])),certificateSha1:certificate,appIdentifier:config.identifier,imeIdentifier:plist.CFBundleIdentifier,
  boundary:'Actual isolated signed Go processes and production Darwin peer proof. No installed IMK dispatch, key export or notarization. Optional vocabulary mode uses the production native handler and synthetic encrypted profile.'});
 const unsigned=path.join(work,'unsigned');
 run('go',['test','-c','-o',unsigned,'./infrastructure/imebridge'],'compile',{cwd:path.join(root,'apps/desktop'),env:{...process.env,GOCACHE:path.join(root,'.workspace/go-cache')}});
 const fixtures=[['app-peer',config.identifier,certificate],['ime-peer',plist.CFBundleIdentifier,certificate],['wrong-peer',`${plist.CFBundleIdentifier}.wrong`,certificate],['adhoc-peer',plist.CFBundleIdentifier,'-']];
 const artifacts=[];
 for(const [name,identifier,signer] of fixtures){
  const file=path.join(work,name);copyFileSync(unsigned,file);
  run('codesign',['--force','--sign',signer,'--identifier',identifier,'--options','runtime','--timestamp=none',file],`sign-${name}`);
  run('codesign',['--verify','--strict',file],`verify-${name}`);
  artifacts.push({name,identifier,developerIDSigned:signer!=='-',sha256:hash(file)});
 }
 const nativeEnv={};
 if(vocabulary){
  const runtime=path.join(root,'apps/desktop/resources/darwin/keyboard');
  const libraryDirectory=path.join(work,'native-lib'),outputDir=path.join(work,'native-build');
  mkdirSync(libraryDirectory);mkdirSync(outputDir);
  const library=path.join(libraryDirectory,'librime.1.dylib');
  const originalLibrary=path.join(runtime,'lib/librime.1.dylib');
  save('native-runtime.json',{originalLibrarySha256:hash(originalLibrary),runtime:JSON.parse(readFileSync(path.join(runtime,'runtime.json')))});
  copyFileSync(originalLibrary,library);
  const signNative=(file,identifier,name)=>{
   run('codesign',['--force','--sign',certificate,'--identifier',identifier,'--options','runtime','--timestamp=none',file],`sign-${name}`);
   run('codesign',['--verify','--strict',file],`verify-${name}`);
   artifacts.push({name:path.relative(work,file),identifier,developerIDSigned:true,sha256:hash(file)});
  };
  signNative(library,`${plist.CFBundleIdentifier}.rime`,'native-library');
  const binary=buildIMETypingFixture({root,outputDir,arch:process.arch==='arm64'?'arm64':'x86_64',fixture:'tests/vocabulary_peer.mm',libraryDirectory,linkArgs:['-Xlinker','-lbsm']});
  signNative(binary,plist.CFBundleIdentifier,'native-peer');
  Object.assign(nativeEnv,{ARION_SIGNED_NATIVE_PEER:binary,ARION_SIGNED_NATIVE_DATA:path.join(runtime,'data'),ARION_SIGNED_NATIVE_LIBRARY:library});
 }
 save('artifacts.json',artifacts);
 const output=run(path.join(work,'app-peer'),[`-test.run=^TestDarwinSigned(PublisherPair|BridgeAdmission${vocabulary?'|NativeVocabulary':''})$`,'-test.v','-test.timeout=90s'],'signed-pairs',{
  env:{...process.env,...nativeEnv,ARION_SIGNED_PEER_FIXTURES:work,ARION_SIGNED_PEER_APP_ID:config.identifier}});
 for(const suite of ['PublisherPair','BridgeAdmission'])for(const name of ['ime-peer','wrong-peer','adhoc-peer'])assert.ok(output.includes(`--- PASS: TestDarwinSigned${suite}/${name}`),`Missing actual signed case ${suite}/${name}`);
 if(vocabulary)for(const name of ['publish','restore-delete','restore-empty'])assert.ok(output.includes(`--- PASS: TestDarwinSignedNativeVocabulary/${name}`),`Missing native vocabulary case ${name}`);
 for(const [file,digest] of Object.entries(JSON.parse(readFileSync(path.join(evidence,'inputs.json'))).sources))assert.equal(hash(path.join(root,file)),digest);
 for(const artifact of artifacts)assert.equal(hash(path.join(work,artifact.name)),artifact.sha256);
 save('status.json',{status:'passed',cases:vocabulary?9:6,nativeVocabulary:vocabulary,productionBridgeAdmission:true,mutualDeveloperID:true,wrongIdentifierRejected:true,adHocRejected:true,exitAndReleaseRejected:true,installedHostAcceptance:false});
 console.log(`Publisher process validation passed: ${evidence}`);
} catch(error){save('status.json',{status:'failed',error:error.message});throw error;}
