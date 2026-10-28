import { buildCandidateFonts } from './lib/windows-candidate-fonts.mjs';
// The production TSF key sink edits an owned ITextStoreACP document. No profile
// registration, global selection, real editor typing or persistent user data.
import {spawn,spawnSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {cpSync,mkdirSync,mkdtempSync,readFileSync,writeFileSync,existsSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import assert from 'node:assert/strict';
import net from 'node:net';
if(process.platform!=='win32') throw new Error('Native Windows is required.');
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const source=path.join(root,'apps/desktop/ime/windows'),common=path.join(source,'../shared');
const sdk=path.join(root,'.workspace/keyboard-runtime/windows/sdk');
const output=path.join(root,'.workspace/tests/windows-tsf-keyboard'); mkdirSync(output,{recursive:true});
const directory=mkdtempSync(path.join(output,'run-'));
const keyboard=path.join(directory,'keyboard');
cpSync(path.join(root,'apps/desktop/resources/windows-amd64/keyboard'),keyboard,{recursive:true});
const hash=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
const report={schema:1,boundary:'Manually dispatch production COM key sink into actual owned ACP TSF edit sessions. Test-only bypass permits foreground Advise rejection for a manually created client; does not verify system key dispatch or installed registration.',directory,cases:[],commands:[],status:'running'};
// Bind evidence to compilation units and their local transitive includes.
// Unrelated registration/loader tests may be edited concurrently.
const sourceSet=new Set([fileURLToPath(import.meta.url)]);
function includeSource(file){
  file=path.resolve(file);if(sourceSet.has(file))return;sourceSet.add(file);
  for(const match of readFileSync(file,'utf8').matchAll(/^\s*#\s*include\s*"([^"]+)"/gm)){
    const dependency=path.resolve(path.dirname(file),match[1]);if(existsSync(dependency))includeSource(dependency);
  }
}
for(const name of ['service.cpp','textstore_fixture.cpp','native_keys_test.cpp','candidate_window_test.cpp','keyboard_runtime.cpp'])includeSource(path.join(source,name));
for(const name of ['pinyin.c','input.c','menu.c','learning.cpp','learning_rime.cpp','learning_windows.cpp','ranking.cpp','ranking_unicode.cpp','emoji.cpp', 'english.cpp', 'english_lexicon.cpp'])includeSource(path.join(common,name));
const sources=[...sourceSet];
report.sources=Object.fromEntries(sources.map(file=>[path.relative(root,file),hash(file)]));
const runtimeSources=[path.join(root,'apps/desktop/resources/windows-amd64/keyboard/runtime.json'),path.join(root,'apps/desktop/resources/windows-amd64/keyboard/lib/ArionRime.dll'),path.join(sdk,'lib/libArionRime.dll.a')];
report.runtimeSources=Object.fromEntries(runtimeSources.map(file=>[path.relative(root,file),hash(file)]));
const save=()=>writeFileSync(path.join(directory,'report.json'),JSON.stringify(report,null,2)+'\n');
function run(command,args){
  const result=spawnSync(command,args,{cwd:root,encoding:'utf8',windowsHide:true,timeout:180000});
  report.commands.push({command,args,status:result.status,stdout:result.stdout,stderr:result.stderr,error:result.error?.message}); save();
  process.stdout.write(result.stdout??''); process.stderr.write(result.stderr??'');
  if(result.error)throw result.error;
  if(result.status!==0)throw new Error(`${command} exited ${result.status}`);
}
let fixture,server;
const connections=new Set();
try{
  const cc=process.env.CC??'gcc',cxx=process.env.CXX??'g++';
  const objects=['pinyin','input','menu'].map(name=>{
    const object=path.join(directory,`${name}.o`);
    run(cc,['-std=c11','-O2','-Wall','-Wextra','-Werror','-I',path.join(sdk,'include'),'-c',path.join(common,`${name}.c`),'-o',object]);return object;
  });
  const base=['-std=c++17','-DINITGUID','-O2','-Wall','-Wextra','-Werror','-static','-isystem',path.join(sdk,'include')];
  const fontResource=buildCandidateFonts(root,cxx,directory,run);
  for(const name of ['native_keys','candidate_window']){
    const binary=path.join(directory,`${name}-test.exe`);
    run(cxx,[...base,path.join(source,`${name}_test.cpp`),...(name==='native_keys'?[objects[1]]:[fontResource]),'-o',binary,'-lgdi32','-ldwrite','-ld2d1','-lole32','-luuid']);
    run(binary,name==='candidate_window'?[directory]:[]);
  }
  const module=path.join(keyboard,'lib/ArionKeyboard.dll');
  run(cxx,[...base,'-DRIME_IMPORTS','-shared',path.join(source,'keyboard_runtime.cpp'),...['learning','learning_rime','learning_windows','ranking','ranking_unicode','emoji','english','english_lexicon'].map(name=>path.join(common,`${name}.cpp`)),objects[0],path.join(sdk,'lib/libArionRime.dll.a'),'-o',module,'-lcrypt32','-lole32','-luuid','-ladvapi32','-ldwrite','-ld2d1']);
  const manifestPath=path.join(keyboard,'runtime.json'),manifest=JSON.parse(readFileSync(manifestPath,'utf8'));
  manifest.files['lib/ArionKeyboard.dll']={sha256:hash(module),sizeBytes:readFileSync(module).length};
  writeFileSync(manifestPath,JSON.stringify(manifest,null,2)+'\n');
  const library=path.join(directory,'ArionTSF.dll');
  run(cxx,[...base,'-DARION_TSF_TESTING','-DARION_TSF_MANUAL_KEYS',`-DARION_KEYBOARD_MANIFEST_SHA256="${hash(manifestPath)}"`,'-shared',path.join(source,'service.cpp'),...objects,fontResource,'-o',library,'-lole32','-loleaut32','-luuid','-ladvapi32','-lshell32','-lgdi32','-ldwrite','-ld2d1']);
  const editor=path.join(directory,'textstore-fixture.exe');
  run(cxx,[...base,'-DARION_TSF_MANUAL_KEYS','-municode',path.join(source,'textstore_fixture.cpp'),'-o',editor,'-lole32','-loleaut32','-luuid','-lgdi32','-ladvapi32']);
  report.identities={service:hash(library),module:hash(module),engine:hash(path.join(keyboard,'lib/ArionRime.dll')),manifest:hash(manifestPath)}; save();
  const pipe=`\\\\.\\pipe\\Arion.TSF.Test.${path.basename(directory)}.${process.pid}`;
  let currentActivation='',peer=null;
  const messages=[];
  server=net.createServer(socket=>{connections.add(socket);peer=socket;let pending='';socket.on('data',chunk=>{pending+=chunk;let end;while((end=pending.indexOf('\n'))>=0){const message=JSON.parse(pending.slice(0,end));pending=pending.slice(end+1);messages.push(message);if(message.op==='activate')currentActivation=message.activation;}});socket.on('close',()=>connections.delete(socket));socket.on('error',()=>{});});
  await new Promise((resolve,reject)=>{server.once('error',reject);server.listen(pipe,resolve);});
  fixture=spawn(editor,[library],{cwd:directory,windowsHide:false,env:{...process.env,ARION_TSF_TEST_USER_DIRECTORY:path.join(directory,'profile'),ARION_TSF_TEST_PIPE:pipe}});
  const lines=[],waiters=[];let pending='',stderr='',ended=null;
  function fail(error){ended=error;while(waiters.length)waiters.shift().reject(error);}
  fixture.on('error',fail);
  const exited=new Promise(resolve=>fixture.once('exit',(code,signal)=>{fail(new Error(`Fixture exited ${code??signal}: ${stderr}`));resolve({code,signal});}));
  fixture.stdout.on('data',chunk=>{pending+=chunk;let end;while((end=pending.indexOf('\n'))>=0){const line=pending.slice(0,end).trim();pending=pending.slice(end+1);if(!line)continue;try{const next=JSON.parse(line);if(waiters.length)waiters.shift().resolve(next);else lines.push(next);}catch(error){fail(error);}}});
  fixture.stderr.on('data',chunk=>{stderr+=chunk;});
  const receive=()=>{
    if(lines.length)return Promise.resolve(lines.shift());
    if(ended)return Promise.reject(ended);
    return new Promise((resolve,reject)=>{
      const waiter={resolve:value=>{clearTimeout(timer);resolve(value);},reject:error=>{clearTimeout(timer);reject(error);}};
      const timer=setTimeout(()=>{const index=waiters.indexOf(waiter);if(index>=0)waiters.splice(index,1);reject(new Error(`Fixture response timed out: ${stderr}`));},10000);
      waiters.push(waiter);
    });
  };
  assert.equal((await receive()).ready,true);
  const command=async text=>{fixture.stdin.write(text+'\n');return receive();};
  const key=async(code,modifiers=0,up=false)=>{const result=await command(`key:${code}:${modifiers}${up?':up':''}`);report.lastKey={code,result,stderr};save();return result;};
  const type=async value=>{for(const letter of value)assert.equal((await key(letter.toUpperCase().charCodeAt(0))).eaten,true,`key ${letter}`);};
  const choose=async text=>{
    for(let attempt=0;attempt<20;attempt++){
      let menu=await command('menu');const index=menu.strings.indexOf(text);
      if(index>=0){const page=Math.floor(index/9);while(menu.page<page){await key(34);menu=await command('menu');}while(menu.page>page){await key(33);menu=await command('menu');}assert.equal((await key(49+index%9)).eaten,true);return;}
      await key(34);
    }
    throw new Error(`Real engine candidate not found: ${text}`);
  };
  const document=async()=>String((await command('text')).text);
  const pause=milliseconds=>new Promise(resolve=>setTimeout(resolve,milliseconds));
  const waitMessage=async(predicate,after=0)=>{const deadline=Date.now()+3000;while(Date.now()<deadline){const found=messages.slice(after).find(predicate);if(found)return found;await pause(10);}throw new Error('Native bridge message timed out');};
  const begin=async session=>{await pause(80);assert.ok(peer&&currentActivation);const after=messages.length;peer.write(JSON.stringify({op:'begin',activation:currentActivation,session})+'\n');return waitMessage(message=>message.op==='begun'&&message.session===session,after);};
  const voice=(session,op,text='',revision=0)=>peer.write(JSON.stringify({op,activation:currentActivation,session,text,revision})+'\n');
  async function test(name,action){await action();report.cases.push({name,status:'passed'});save();console.log(`PASS ${name}`);}
  await test('Native hello announces private-password-safe protocol 2',async()=>{const hello=await waitMessage(message=>message.op==='hello');assert.equal(hello.protocol,2);});
  const settingsFile=path.join(directory,'profile/keyboard.json');
  const contextFile=path.join(directory,'profile/learning/arion_context.arion-userdb');
  const contextBytes=()=>existsSync(contextFile)?readFileSync(contextFile):null;
  await test('Chinese mode offers English words and commits their displayed identity through TSF',async()=>{
    for(const word of ['artificial','hello','computer','software','can','an','you','hi','bailing','jingling']){
      await command('reset');await type(word);const menu=await command('menu');
      const index=menu.strings.indexOf(word);assert.ok(index>=0 && index<5,JSON.stringify(menu));
      assert.equal(menu.visible,true);await key(49+index);assert.equal(await document(),`A🙂${word}Z`);
      await pause(80);assert.equal(await document(),`A🙂${word}Z`);
    }
    await command('reset');await type('artificial');await key(32);assert.equal(await document(),'A🙂artificialZ');
    await command('reset');await type('artificial');await choose('Artificial');assert.equal(await document(),'A🙂ArtificialZ');
    await command('reset');await type('artificial');await command('cacheclick');await key(27);await command('staleclick');assert.equal(await document(),'A🙂中Z');
  });
  await test('Typed native Emoji has protected text slots and commits exactly once without Chinese context learning',async()=>{
    await command('reset');await type('xiaolian');const menu=await command('menu');
    assert.equal(menu.visible,true);const index=menu.strings.indexOf('😊');assert.ok(index>=3 && index<9,JSON.stringify(menu));
    const before=contextBytes();await key(49+index);assert.equal(await document(),'A🙂😊Z');await pause(350);assert.equal(await document(),'A🙂😊Z');
    assert.deepEqual(contextBytes(),before,'Emoji commit must not rewrite the Chinese context store');
  });
  await test('A selected Chinese prefix and Emoji suffix commit together without Chinese context learning',async()=>{
    await command('reset');await type('nixiaolian');await choose('你');const menu=await command('menu');assert.ok(menu.strings.includes('😊'),JSON.stringify(menu));
    const before=contextBytes();await choose('😊');assert.equal(await document(),'A🙂你😊Z');await pause(350);assert.deepEqual(contextBytes(),before);
  });
  const highlightEmoji=async()=>{
    let menu=await command('menu');const index=menu.strings.indexOf('😊');assert.ok(index>=3 && index<9);
    await key(34);await key(33);menu=await command('menu');assert.equal(menu.page,0);
    for(let i=0;i<index;i++)await key(39);
    assert.equal((await command('menu')).selection,index);
  };
  await test('Space and engine punctuation commit the displayed Emoji identity',async()=>{
    await command('reset');await type('xiaolian');await highlightEmoji();await key(32);assert.equal(await document(),'A🙂😊Z');
    await command('reset');await type('xiaolian');await highlightEmoji();await key(190);assert.equal(await document(),'A🙂😊。Z');
  });
  await test('Middle caret suppresses Emoji and End restores it without changing the owned draft',async()=>{
    await command('reset');await type('xiaolian');assert.ok((await command('menu')).strings.includes('😊'));
    await key(36);assert.equal((await command('menu')).strings.includes('😊'),false);
    await key(35);assert.ok((await command('menu')).strings.includes('😊'));await key(27);assert.equal(await document(),'A🙂中Z');
  });
  await test('Emoji preference refresh preserves text order and revokes canceled candidates',async()=>{
    await command('reset');await type('xiaolian');const enabled=(await command('menu')).strings.filter(text=>text!=='😊').slice(0,27);
    writeFileSync(settingsFile,JSON.stringify({emojiSuggestions:false}));await pause(350);
    const disabled=await command('menu');assert.equal(disabled.strings.includes('😊'),false);assert.deepEqual(disabled.strings.slice(0,27),enabled);
    await key(27);writeFileSync(settingsFile,JSON.stringify({emojiSuggestions:true}));await pause(350);await type('xiaolian');assert.ok((await command('menu')).strings.includes('😊'));
    await command('cacheclick');await key(27);await command('staleclick');assert.equal(await document(),'A🙂中Z');
    await type('nihao');assert.equal((await command('menu')).strings.includes('😊'),false);await key(27);
  });
  await test('Real Pinyin nihao commits Chinese exactly once',async()=>{await command('reset');await type('nihao');assert.match(await document(),/^A🙂ni.?haoZ$/u);assert.equal((await key(32)).eaten,true);assert.equal(await document(),'A🙂你好Z');await new Promise(resolve=>setTimeout(resolve,350));assert.equal(await document(),'A🙂你好Z');});
  await test('Keyboard-focus receipt rebinds document without a new thread-focus callback',async()=>{await command('reset');await command('threadblur');assert.equal((await key(78)).eaten,false);await command('keyboardfocus');await type('nihao');await key(32);assert.equal(await document(),'A🙂你好Z');});
  await test('Chromium transitory editable store accepts Pinyin while protected fields remain refused',async()=>{await command('reset');await command('chromium');await type('nihao');await key(32);assert.equal(await document(),'A🙂你好Z');await command('reset');await command('password');assert.equal((await key(78)).eaten,false);await command('edit');await command('readonly');assert.equal((await key(78)).eaten,false);await command('writable');await command('regularstore');});
  await test('Transitory voice ownership requires explicit non-password scope',async()=>{
    await command('reset');await command('chromium');await begin('chromium-voice');voice('chromium-voice','mark','voice draft',1);await pause(80);assert.equal(await document(),'A🙂voice draftZ');const after=messages.length;voice('chromium-voice','commit','voice final',2);await waitMessage(message=>message.op==='committed'&&message.session==='chromium-voice',after);assert.equal(await document(),'A🙂voice finalZ');
    await command('reset');await command('noscope');assert.equal((await key(78)).eaten,false);assert.equal(await document(),'A🙂中Z');const rejected=messages.length;await pause(80);peer.write(JSON.stringify({op:'begin',activation:currentActivation,session:'no-scope'})+'\n');await waitMessage(message=>message.op==='escape'&&message.session==='no-scope',rejected);assert.equal(messages.slice(rejected).some(message=>message.op==='begun'&&message.session==='no-scope'),false);await command('scope');await command('regularstore');
  });
  await test('Escape restores originally selected Unicode text',async()=>{await command('reset');await type('ni');assert.equal((await key(27)).eaten,true);assert.equal(await document(),'A🙂中Z');});
  await test('Private Chromium password and PIN scopes refuse keys and voice ownership',async()=>{
    await command('reset');await command('chromium');
    for(const scope of [31,61,63,64,65,66]){await command(`inputscope:${scope}`);assert.equal((await key(78)).eaten,false,`scope ${scope}`);assert.equal(await document(),'A🙂中Z');await pause(80);const after=messages.length,session=`secret-scope-${scope}`;peer.write(JSON.stringify({op:'begin',activation:currentActivation,session})+'\n');await waitMessage(message=>message.op==='escape'&&message.session===session,after);assert.equal(messages.slice(after).some(message=>message.op==='begun'&&message.session===session),false);assert.equal(await document(),'A🙂中Z');}
    await command('inputscope:0');await command('regularstore');await type('ni');await key(27);
  });
  await test('Password and read-only contexts refuse engine ownership',async()=>{await command('reset');await command('password');assert.equal((await key(78)).eaten,false);await command('edit');await command('readonly');assert.equal((await key(78)).eaten,false);assert.equal(await document(),'A🙂中Z');await command('writable');});
  await test('Host composition refusal does not consume a key',async()=>{await command('reset');await command('refuse');assert.equal((await key(78)).eaten,false);assert.equal(await document(),'A🙂中Z');await command('accept');});
  await test('Engine does not process a key before a granted write lock',async()=>{await command('reset');await command('block');assert.equal((await key(78)).eaten,false);await command('unblock');await type('i');assert.equal(await document(),'A🙂iZ');await key(27);});
  await test('User replacement survives stale composition cleanup',async()=>{await command('reset');await type('ni');await command('replace');await new Promise(resolve=>setTimeout(resolve,350));assert.equal(await document(),'A🙂userZ');});
  await test('Focus loss restores only unchanged marked text',async()=>{await command('reset');await type('ni');await command('password');await new Promise(resolve=>setTimeout(resolve,350));assert.equal(await document(),'A🙂中Z');await command('edit');});
  await test('Zero-width caret anchors an actual nonactivating candidate window',async()=>{await command('reset');await command('zerocaret');await type('ni');const menu=await command('menu');assert.equal(menu.visible,true);assert.ok(menu.strings.length>=9);assert.ok(menu.width>0&&menu.height>0);await key(27);await command('caret');});
  await test('Delayed browser layout restores each current candidate revision through read-only sessions',async()=>{
    await command('reset');await command('zerocaret');await command('delaylayout');
    for(const letter of 'nihao'){
      await type(letter);await pause(40);let menu=await command('menu');assert.equal(menu.visible,false,`missing layout at ${letter}`);assert.ok(menu.strings.length>0);
      const before=await command('layoutstate');assert.ok(before.failures>0);const text=await document();
      await command('readlocks');await command('layout');await pause(50);menu=await command('menu');const after=await command('layoutstate');
      assert.equal(menu.visible,true,`layout-ready at ${letter}`);assert.equal(await document(),text);assert.equal(after.writes,before.writes);assert.equal(after.writeExtents,before.writeExtents);assert.ok(after.reads>before.reads);
      await command('writelocks');
    }
    await key(32);assert.equal(await document(),'A🙂你好Z');await command('instantlayout');await command('caret');
  });
  await test('Layout changes reposition or suppress candidates without stale coordinates',async()=>{
    await command('reset');await type('ni');let menu=await command('menu');assert.equal(menu.visible,true);const x=menu.x;
    await command('scroll');await pause(40);menu=await command('menu');assert.equal(menu.visible,true);assert.equal(menu.x,x+40);
    await command('clipped');await pause(40);assert.equal((await command('menu')).visible,false);
    await command('unclipped');await pause(40);assert.equal((await command('menu')).visible,true);
    await command('nolayout');await command('layout');await pause(30);await key(27);await command('layout');await pause(40);assert.equal((await command('menu')).visible,false);assert.equal(await document(),'A🙂中Z');
  });
  await test('Deferred layout cannot resurrect a canceled or focus-revoked composition',async()=>{
    await command('reset');await command('delaylayout');await type('ni');await command('block');await command('layout');await pause(30);await command('keyboardblur');await command('unblock');await pause(40);assert.equal((await command('menu')).visible,false);assert.equal(await document(),'A🙂中Z');
    await command('keyboardfocus');await command('instantlayout');await type('ni');assert.equal((await command('menu')).visible,true);await key(27);await command('layout');await pause(30);assert.equal((await command('menu')).visible,false);
  });
  await test('Number selection commits the TSF accessible candidate ordering',async()=>{await command('reset');await type('ni');const menu=await command('menu');assert.equal((await key(50)).eaten,true);assert.equal(await document(),`A🙂${menu.strings[1]}Z`);});
  await test('Expanded six-row native menu selects another row by mouse',async()=>{await command('reset');await type('ni');await key(34);const menu=await command('menu');assert.equal(menu.page,1);assert.ok(menu.height>32*2);await command('click:1');assert.equal(await document(),`A🙂${menu.strings[9]}Z`);});
  await test('Hidden stale native candidate click cannot insert',async()=>{await command('reset');await type('ni');await command('cacheclick');await key(27);await command('staleclick');assert.equal(await document(),'A🙂中Z');});
  await test('Return commits raw Pinyin rather than a candidate display string',async()=>{await command('reset');await type('nihao');await key(13);assert.equal(await document(),'A🙂nihaoZ');});
  await test('Partial prefix remains provisional until suffix confirmation',async()=>{await command('reset');await type('nihao');await choose('你');assert.match(await document(),/^A🙂你.*haoZ$/u);await key(32);assert.equal(await document(),'A🙂你好Z');});
  await test('Backspace rolls a selected prefix back into raw composition',async()=>{await command('reset');await type('nihao');await choose('你');const selected=await document();await key(8);assert.notEqual(await document(),selected);await choose('你好');assert.equal(await document(),'A🙂你好Z');});
  await test('Home End and middle edits preserve the original suffix',async()=>{await command('reset');await type('nihao');await key(36);assert.equal((await command('state')).start,3);await key(35);assert.equal((await command('state')).start,(await document()).length-1);await key(37);await key(8);await type('i');assert.equal((await document()).replaceAll(' ',''),'A🙂nihaoZ');await key(35);await key(32);assert.equal(await document(),'A🙂你好Z');});
  await test('Forward Delete at Home removes only the engine caret character',async()=>{await command('reset');await type('nihao');await key(36);await key(46);assert.equal((await document()).replaceAll(' ',''),'A🙂ihaoZ');await type('n');await key(35);await key(32);assert.equal(await document(),'A🙂你好Z');});
  await test('Short Shift switches ASCII and another tap restores Chinese',async()=>{await command('reset');await key(160,1);assert.equal((await key(160,0,true)).eaten,true);assert.equal((await command('mode')).mode,'En');assert.equal((await key(78)).eaten,false);await key(160,1);await key(160,0,true);assert.equal((await command('mode')).mode,'中');await type('ni');await key(27);});
  await test('Control and Super shortcuts cancel ownership and pass through',async()=>{for(const modifiers of [4,1<<26]){await command('reset');await type('ni');assert.equal((await key(65,modifiers)).eaten,false);assert.equal(await document(),'A🙂中Z');}});
  await test('Interrupted Shift tap does not toggle Chinese mode',async()=>{await command('reset');await key(160,1);await key(17,5);await key(160,0,true);assert.equal((await command('mode')).mode,'中');});
  await test('Duplicate Shift notifications retain one original tap deadline',async()=>{await command('reset');await key(160,1);await pause(40);await key(160,1);await key(160,0,true);assert.equal((await command('mode')).mode,'En');await key(160,0,true);assert.equal((await command('mode')).mode,'En');await key(160,1);await pause(320);await key(160,1);await key(160,0,true);assert.equal((await command('mode')).mode,'En');await key(160,1);await key(160,0,true);assert.equal((await command('mode')).mode,'中');});
  await test('Unambiguous anonymous Shift transitions use the shared tap state',async()=>{await key(0,1);await key(255,0,true);assert.equal((await command('mode')).mode,'En');await key(0,1);await key(255,0,true);assert.equal((await command('mode')).mode,'中');});
  await test('Native full-width and punctuation choices persist exact shared options',async()=>{await command('reset');assert.equal((await command('action:3')).hr,0);let options=JSON.parse(readFileSync(path.join(directory,'profile/options.json'),'utf8'));assert.equal(options.fullWidth,true);assert.equal(options.englishPunctuation,false);await command('action:3');await key(190,4);options=JSON.parse(readFileSync(path.join(directory,'profile/options.json'),'utf8'));assert.equal(options.fullWidth,false);assert.equal(options.englishPunctuation,true);await key(190,4);});
  await test('Unsaved mode choice keeps existing draft and leaves no new ownership',async()=>{
    await command('reset');const before=await command('state');const bytes=readFileSync(path.join(directory,'profile/options.json'),'utf8');assert.equal((await command('lockoptions')).locked,true);await command('action:3');let after=await command('state');assert.equal(after.text,before.text);assert.equal(after.starts-after.ends,before.starts-before.ends);await command('unlockoptions');
    await type('ni');const pending=await command('state');await command('lockoptions');await command('action:3');after=await command('state');assert.equal(after.text,pending.text);assert.equal(after.starts,pending.starts);assert.equal(after.ends,pending.ends);await command('unlockoptions');assert.equal(readFileSync(path.join(directory,'profile/options.json'),'utf8'),bytes);await key(27);
  });
  await test('Caret badge is 23 DIP, click-through, keeps focus and expires after one second',async()=>{await command('reset');await key(32,1);let badge=await command('badge');assert.equal(badge.visible,true);assert.equal(badge.width,Math.round(23*badge.dpi/96));assert.equal(badge.height,badge.width);assert.equal(badge.hit,-1);assert.equal(badge.focusRetained,true);await pause(1050);assert.equal((await command('badge')).visible,false);await key(32,1);await type('n');assert.equal((await command('badge')).visible,false);await key(27);});
  await test('Missing caret suppresses the badge while retaining the mode switch',async()=>{await command('reset');await command('nocaret');await key(32,1);assert.equal((await command('badge')).visible,false);assert.equal((await command('mode')).mode,'En');await key(32,1);await command('caret');});
  await test('Width and punctuation affect actual commits while ASCII passes original events',async()=>{await command('reset');await command('action:3');await key(49);assert.equal(await document(),'A🙂１Z');await command('reset');await key(32);assert.equal(await document(),'A🙂　Z');await command('reset');await key(190);assert.equal(await document(),'A🙂。Z');await command('reset');await key(190,4);await key(190);assert.equal(await document(),'A🙂．Z');await command('reset');await key(32,1);const bytes=readFileSync(path.join(directory,'profile/options.json'),'utf8');assert.equal((await key(190,4)).eaten,false);assert.equal((await key(49)).eaten,false);assert.equal(await document(),'A🙂中Z');assert.equal(readFileSync(path.join(directory,'profile/options.json'),'utf8'),bytes);await key(32,1);await command('action:3');await key(190,4);});
  await test('Reset while marked restores original bytes and recreates sessions',async()=>{await command('reset');await type('ni');assert.equal((await command('action:6')).hr,0);await pause(350);assert.equal(await document(),'A🙂中Z');await type('nihao');await key(32);assert.equal(await document(),'A🙂中你好Z');});
  await test('Physical typing revokes voice and rejects its late commit',async()=>{await command('reset');await begin('voice-to-keyboard');voice('voice-to-keyboard','mark','voice draft',1);await pause(80);assert.equal(await document(),'A🙂voice draftZ');await type('n');const pending=await document();assert.equal(pending,'A🙂中nZ');voice('voice-to-keyboard','commit','LATE',2);await pause(100);assert.equal(await document(),pending);await key(27);assert.equal(await document(),'A🙂中Z');});
  await test('Voice takeover cancels typing before acquiring its own range',async()=>{await command('reset');await type('ni');await begin('keyboard-to-voice');voice('keyboard-to-voice','mark','voice',1);await pause(80);assert.equal(await document(),'A🙂中voiceZ');const after=messages.length;voice('keyboard-to-voice','commit','voice final',2);await waitMessage(message=>message.op==='committed'&&message.session==='keyboard-to-voice',after);assert.equal(await document(),'A🙂中voice finalZ');});
  await test('Keyboard-only focus loss restores drafts and refuses keys or voice until receipt',async()=>{
    await command('reset');await type('ni');await command('keyboardblur');await command('keyboardblur');await pause(80);assert.equal(await document(),'A🙂中Z');assert.equal((await key(78)).eaten,false);await command('keyboardfocus');await begin('keyboard-blur-voice');voice('keyboard-blur-voice','mark','voice draft',1);await pause(80);assert.equal(await document(),'A🙂中voice draftZ');const after=messages.length;await command('keyboardblur');await pause(100);assert.equal(await document(),'A🙂中Z');assert.equal([...connections].some(socket=>!socket.destroyed),false);voice('keyboard-blur-voice','commit','LATE',2);peer.write(JSON.stringify({op:'begin',activation:currentActivation,session:'blurred-begin'})+'\n');await pause(80);assert.equal(messages.slice(after).some(message=>message.op==='begun'&&message.session==='blurred-begin'),false);assert.equal(await document(),'A🙂中Z');await command('keyboardfocus');await type('ni');await key(27);
  });
  await test('App bridge disconnect cannot erase a keyboard composition',async()=>{await command('reset');await type('ni');const pending=await document();for(const socket of connections)socket.destroy();await new Promise(resolve=>server.close(resolve));server=null;await pause(350);assert.equal(await document(),pending);await key(32);assert.equal(await document(),'A🙂你Z');});
  await test('Saved candidate preferences refresh without the desktop app',async()=>{await command('reset');writeFileSync(path.join(directory,'profile/keyboard.json'),JSON.stringify({pageSize:5,candidateSize:'large',appearance:'dark',highlight:{dark:{fill:'#123456',text:'#ffffff'},light:{fill:'#abcdef',text:'#14110f'}}}));await pause(350);await type('ni');const menu=await command('menu');await key(34);const expanded=await command('menu');report.lastSettingsMenus={before:menu,expanded};save();assert.equal(expanded.page,1);assert.ok(expanded.height>36*2);await key(49);assert.equal(await document(),`A🙂${menu.strings[5]}Z`);});
  fixture.stdin.write('quit\n');
  const exit=await Promise.race([exited,new Promise((_,reject)=>{const timer=setTimeout(()=>reject(new Error('Fixture shutdown timed out')),5000);timer.unref();})]);
  assert.equal(exit.code,0);fixture=null;
  for(const file of sources)assert.equal(hash(file),report.sources[path.relative(root,file)],`source changed during native run: ${file}`);
  for(const file of runtimeSources)assert.equal(hash(file),report.runtimeSources[path.relative(root,file)],`runtime changed during native run: ${file}`);
  report.status='passed';
}catch(error){report.status='failed';report.error=error.stack;throw error;}
finally{if(fixture)fixture.kill();for(const socket of connections)socket.destroy();if(server)server.close();save();console.log(`TSF keyboard evidence: ${path.join(directory,'report.json')}`);}
