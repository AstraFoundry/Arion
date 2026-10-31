// Build/cache the reviewed Rime source without modifying staged product resources.
import {execFileSync,spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {existsSync,mkdirSync,readFileSync,writeFileSync,statSync,lstatSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {isDeepStrictEqual} from 'node:util';
import {buildParallelism} from '../build-parallelism.mjs';
import {pinnedFile} from './pinned-file.mjs';
import {verifiedCorrectionIdentity} from './keyboard-correction-patch.mjs';
const digest=bytes=>createHash('sha256').update(bytes).digest('hex');
const hash=file=>digest(readFileSync(file));
const command=(name,args)=>execFileSync(name,args,{encoding:'utf8'}).trim();
// Cache manifests may not redirect the build to unchecked or external files.
export function verifyMacRuntimeCache({directory,record,identity,version}) {
 const library=`build/lib/librime.${version}.dylib`,deployer='build/bin/rime_deployer';
 if(!isDeepStrictEqual(record.identity,identity)||record.library!==library||record.deployer!==deployer)
  throw new Error('Rime build cache identity or output paths differ.');
 if(!record.outputs||!isDeepStrictEqual(Object.keys(record.outputs).sort(),[library,deployer].sort()))
  throw new Error('Rime build cache output inventory differs.');
 for(const file of [library,deployer]) {
  const item=record.outputs[file],target=path.join(directory,file),stat=lstatSync(target);
  if(!stat.isFile()||!item||!Number.isSafeInteger(item.sizeBytes)||item.sizeBytes<=0||
   !/^[a-f0-9]{64}$/.test(item.sha256)||stat.size!==item.sizeBytes||hash(target)!==item.sha256)
   throw new Error(`Corrupt Rime build cache: ${target}`);
 }
}
export function buildMacKeyboardRuntime({manifest,work,architectures=manifest.engine.architectures,minimumSystemVersion='14.0',applyPatch=true}) {
 if(process.platform!=='darwin')throw new Error('macOS Rime source build requires macOS.');
 if(!architectures.length||architectures.some(a=>!['arm64','x86_64'].includes(a))||new Set(architectures).size!==architectures.length)throw new Error('Invalid Rime build architectures.');
 const sources=Object.fromEntries(['source','dependencies','boost'].map(key=>[key,manifest.archives[key]]));
 const identity={sources,architectures:[...architectures].sort(),minimumSystemVersion,logging:true,externalPlugins:false,
  correctionIdentity:applyPatch?manifest.patches.correctionIdentity:null,
  builderSha256:hash(fileURLToPath(import.meta.url)),patcherSha256:hash(fileURLToPath(new URL('./keyboard-correction-patch.mjs',import.meta.url))),
  toolchain:{clang:digest(command('xcrun',['clang++','--version'])),cmake:digest(command('cmake',['--version'])),sdk:command('xcrun',['--sdk','macosx','--show-sdk-version'])}};
 const fingerprint=digest(JSON.stringify(identity)),directory=path.resolve(work,'source-build',fingerprint),recordFile=path.join(directory,'build.json');
 if(existsSync(recordFile)){
  const record=JSON.parse(readFileSync(recordFile,'utf8'));
  verifyMacRuntimeCache({directory,record,identity,version:manifest.engine.version});
  return {directory,library:path.join(directory,record.library),deployer:path.join(directory,record.deployer),build:{...identity,fingerprint,outputs:record.outputs}};
 }
 if(existsSync(directory))throw new Error(`Incomplete Rime build retained at ${directory}; preserve and move it before retrying.`);
 mkdirSync(directory,{recursive:true});mkdirSync(path.join(directory,'evidence'));
 const save=(name,value)=>writeFileSync(path.join(directory,'evidence',name),JSON.stringify(value,null,2)+'\n');
 const commands=[];
 const run=(name,args,label)=>{
  const result=spawnSync(name,args,{encoding:'utf8',timeout:600000,maxBuffer:32*1024*1024});
  writeFileSync(path.join(directory,'evidence',`${label}.stdout`),result.stdout??'');writeFileSync(path.join(directory,'evidence',`${label}.stderr`),result.stderr??'');
  commands.push({name,args,status:result.status,error:result.error?.message??null});save('commands.json',commands);
  if(result.error||result.status!==0)throw new Error(`Rime ${label} failed; see ${directory}/evidence/${label}.stderr`);
 };
 save('status.json',{status:'building',identity});
 try{
  const archives=Object.fromEntries(Object.entries(sources).map(([key,spec])=>[key,pinnedFile(spec)]));
  for(const name of ['source','deps','boost'])mkdirSync(path.join(directory,name));
  run('tar',['-xf',archives.source,'-C',path.join(directory,'source')],'extract-source');
  run('tar',['-xf',archives.dependencies,'-C',path.join(directory,'deps')],'extract-dependencies');
  run('tar',['-xf',archives.boost,'-C',path.join(directory,'boost'),'boost_1_89_0/boost','boost_1_89_0/LICENSE_1_0.txt'],'extract-boost');
  const source=path.join(directory,'source',`librime-${manifest.engine.commit}`),spelling=path.join(source,'src/rime/algo/syllabifier.cc');
  const original=readFileSync(spelling,'utf8');
  if(hash(spelling)!==manifest.patches.correctionIdentity.originalSha256)throw new Error('Extracted Rime source differs from patch pin.');
  if(applyPatch)writeFileSync(spelling,verifiedCorrectionIdentity(original,manifest.patches.correctionIdentity));
  save('source.json',{originalSha256:digest(original),compiledSha256:hash(spelling)});
  const build=path.join(directory,'build'),deps=path.join(directory,'deps');
  run('cmake',['-S',source,'-B',build,'-DCMAKE_BUILD_TYPE=Release',`-DCMAKE_OSX_ARCHITECTURES=${identity.architectures.join(';')}`,`-DCMAKE_OSX_DEPLOYMENT_TARGET=${minimumSystemVersion}`,
   '-DCMAKE_SKIP_RPATH=ON','-DBUILD_STATIC=ON','-DBUILD_SHARED_LIBS=ON','-DBUILD_TEST=OFF','-DBUILD_DATA=OFF','-DBUILD_MERGED_PLUGINS=OFF','-DENABLE_EXTERNAL_PLUGINS=OFF','-DENABLE_LOGGING=ON','-DALSO_LOG_TO_STDERR=OFF',
   '-DCMAKE_DISABLE_FIND_PACKAGE_Gflags=ON','-DCMAKE_IGNORE_PREFIX_PATH=/opt/homebrew;/usr/local',
   '-DBoost_NO_BOOST_CMAKE=ON','-DBoost_NO_SYSTEM_PATHS=ON',`-DBOOST_ROOT=${directory}/boost/boost_1_89_0`,`-DCMAKE_PREFIX_PATH=${deps}`,`-DCMAKE_INCLUDE_PATH=${deps}/include`,`-DCMAKE_LIBRARY_PATH=${deps}/lib`],'configure');
  const compile=readFileSync(path.join(build,'compile_commands.json'),'utf8');
  if(['/opt/homebrew/include','/usr/local/include'].some(prefix=>compile.includes(prefix)))throw new Error('Rime build selected unpinned system headers.');
  const jobs=String(buildParallelism());
  console.log(`ci-build-jobs: ${jobs}`);
  run('cmake',['--build',build,'--target','rime','rime_deployer','--parallel',jobs],'compile');
  const library=`build/lib/librime.${manifest.engine.version}.dylib`,deployer='build/bin/rime_deployer';
  const actual=command('lipo',['-archs',path.join(directory,library)]).split(/\s+/).sort();
  if(!isDeepStrictEqual(actual,identity.architectures))throw new Error('Built Rime architectures differ.');
  run('otool',['-arch','all','-L',path.join(directory,library)],'linkage');
  run('otool',['-arch','all','-l',path.join(directory,library)],'load-commands');
  if(readFileSync(path.join(directory,'evidence/load-commands.stdout'),'utf8').includes('cmd LC_RPATH'))throw new Error('Source-built Rime contains an unexpected runtime search path.');
  const dependencies=[...readFileSync(path.join(directory,'evidence/linkage.stdout'),'utf8').matchAll(/^\s+(\S+) \(compatibility/gm)].map(m=>m[1]);
  if(dependencies.length!==3*actual.length||dependencies.some(d=>!['@rpath/librime.1.dylib','/usr/lib/libc++.1.dylib','/usr/lib/libSystem.B.dylib'].includes(d)))throw new Error('Unexpected source-built Rime linkage.');
  const outputs=Object.fromEntries([library,deployer].map(file=>[file,{sha256:hash(path.join(directory,file)),sizeBytes:statSync(path.join(directory,file)).size}]));
  const record={identity,library,deployer,outputs};writeFileSync(recordFile,JSON.stringify(record,null,2)+'\n');save('status.json',{status:'completed',fingerprint});
  return {directory,library:path.join(directory,library),deployer:path.join(directory,deployer),build:{...identity,fingerprint,outputs}};
 }catch(error){save('status.json',{status:'failed',error:error.message});throw error;}
}
