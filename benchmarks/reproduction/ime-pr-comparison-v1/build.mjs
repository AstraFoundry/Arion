import {execFileSync} from 'node:child_process';
import {mkdirSync,writeFileSync,readFileSync,readdirSync,statSync} from 'node:fs';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {pathToFileURL,fileURLToPath} from 'node:url';
const [rootArg,outputArg]=process.argv.slice(2);
if(!rootArg||!outputArg)throw new Error('usage: build.mjs SNAPSHOT_ROOT NEW_BUILD_DIR');
const root=path.resolve(rootArg),output=path.resolve(outputArg);
mkdirSync(output,{recursive:false});
const {buildIMELearning}=await import(pathToFileURL(path.join(root,'scripts/lib/build-ime-learning.mjs')));
const learning=buildIMELearning({root,outputDir:output,arch:'arm64',testing:true});
const runtime=path.join(root,'apps/desktop/resources/darwin/keyboard'),native=path.join(root,'apps/desktop/ime/macos');
const driver=fileURLToPath(new URL('./driver.mm',import.meta.url));
const objects=[];
for(const [i,file] of ['keyboard.m','ranking.m','typing.m',driver].entries()){
 const objcxx=i===3,object=path.join(output,`native-${i}.o`);
 execFileSync('xcrun',[objcxx?'clang++':'clang',...(objcxx?['-std=c++17']:[]),'-target',learning.target,'-O2','-fobjc-arc','-Wall','-Wextra','-Werror','-DARION_LEARNING_TESTING','-DGLOG_USE_GLOG_EXPORT','-I',native,'-I',path.join(runtime,'include'),'-isystem',path.join(root,'.workspace/keyboard-runtime/sdk/include'),'-c',objcxx?file:path.join(native,file),'-o',object],{stdio:'inherit'});objects.push(object);
}
const binary=path.join(output,'driver');
execFileSync('xcrun',[learning.linker,'-target',learning.target,...objects,...learning.objects,path.join(runtime,'lib/librime.1.dylib'),'-Xlinker','-rpath','-Xlinker',path.join(runtime,'lib'),'-framework','Cocoa','-framework','InputMethodKit',...learning.linkArgs,'-o',binary],{stdio:'inherit'});
const hash=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const files={};
function inventory(directory){for(const name of readdirSync(directory)){const file=path.join(directory,name);if(statSync(file).isDirectory())inventory(file);else files[path.relative(root,file)]={sha256:hash(file),sizeBytes:statSync(file).size};}}
inventory(path.join(root,'apps/desktop/ime'));inventory(runtime);inventory(path.join(root,'scripts/lib'));
writeFileSync(path.join(output,'identity.json'),JSON.stringify({root,binary,sha256:hash(binary),driverSha256:hash(driver),files,flags:['-O2','arm64-apple-macos14.0','ARION_LEARNING_TESTING'],architecture:execFileSync('lipo',['-archs',binary],{encoding:'utf8'}).trim()},null,2)+'\n');
console.log(binary);
