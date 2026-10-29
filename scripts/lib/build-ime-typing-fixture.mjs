// One native typing/AppKit fixture build, shared by interaction and IPC checks.
import {execFileSync} from 'node:child_process';
import path from 'node:path';
import {buildIMELearning} from './build-ime-learning.mjs';
export function buildIMETypingFixture({root,outputDir,arch,fixture,libraryDirectory,linkArgs=[]}) {
 const source=path.join(root,'apps/desktop/ime/macos'),runtime=path.join(root,'apps/desktop/resources/darwin/keyboard');
 const learning=buildIMELearning({root,outputDir,arch,testing:true});
 const objects=['keyboard.m','ranking.m','typing.m','candidate_panel.m',fixture].map((file,i)=>{
  const object=path.join(outputDir,`typing-${i}.o`),test=i===4;
  execFileSync('xcrun',[test?'clang++':'clang',...(test?['-x','objective-c++','-std=c++17']:[]),
   '-target',learning.target,'-fobjc-arc','-Wall','-Wextra','-Werror','-DARION_LEARNING_TESTING','-DGLOG_USE_GLOG_EXPORT',
   '-I',path.join(runtime,'include'),'-isystem',path.join(root,'.workspace/keyboard-runtime/sdk/include'),
   '-c',path.join(source,file),'-o',object],{cwd:root,stdio:'inherit'});
  return object;
 });
 const library=libraryDirectory??path.join(runtime,'lib'),binary=path.join(outputDir,'typing-test');
 execFileSync('xcrun',[learning.linker,'-target',learning.target,...objects,...learning.objects,
  path.join(library,'librime.1.dylib'),'-Xlinker','-rpath','-Xlinker',library,
  '-framework','Cocoa','-framework','QuartzCore','-framework','InputMethodKit',...learning.linkArgs,...linkArgs,'-o',binary],{cwd:root,stdio:'inherit'});
 return binary;
}
