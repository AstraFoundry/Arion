import {spawn,execFileSync} from 'node:child_process';
import {createInterface} from 'node:readline';
import {existsSync,mkdirSync,readFileSync,writeFileSync,rmSync,readdirSync,lstatSync,readlinkSync} from 'node:fs';
import {createHash} from 'node:crypto';
import path from 'node:path';
import net from 'node:net';
import os from 'node:os';
const root='/Users/steven/code/Arion';
const base=path.join(root,'.workspace/e2e/imk-system');
const run=path.join(base,'run-14');
if(existsSync(path.join(run,'state.json'))) throw new Error('Existing run requires review, not overwrite');
mkdirSync(run,{recursive:true});
const sources=path.join(base,'sources');
const probe=path.join(root,'.workspace/research/imk-alignment/arion-ime-probe');
const bundle=path.join(os.homedir(),'Library/Input Methods/ArionProfileValidation.app');
if(existsSync(bundle))throw new Error('Fresh diagnostic input-source path already exists');
const staged=path.join(run,'payload/ArionInput.app');
mkdirSync(path.dirname(staged),{recursive:true});
execFileSync('ditto',[path.join(root,'apps/desktop/build/darwin-arm64/Arion.app/Contents/Resources/ArionInput.app'),staged]);
const backup=path.join(run,'original/ArionInput.app');
const state={phase:'preparing',baseline:JSON.parse(execFileSync(sources,[],{encoding:'utf8'})),bundle,backup,original:existsSync(bundle),profile:path.join(run,'profile'),fixtureSocket:'/tmp/arion-ime-fixture-01.sock'};
if(state.baseline.own.some(source=>source.enabled||source.selected))throw new Error('Existing active Arion source requires review');
let host,fixture,finished=false;
const execute=(cmd,args)=>execFileSync(cmd,args,{encoding:'utf8'}).trim();
const save=()=>writeFileSync(path.join(run,'state.json'),JSON.stringify(state,null,2)+'\n');
const inventory=directory=>{
 const out={};
 const walk=dir=>{for(const name of readdirSync(dir)){const file=path.join(dir,name),rel=path.relative(directory,file),info=lstatSync(file);if(info.isSymbolicLink())out[rel]={link:readlinkSync(file)};else if(info.isDirectory())walk(file);else out[rel]={mode:info.mode&0o777,sha256:createHash('sha256').update(readFileSync(file)).digest('hex')};}};
 walk(directory);return out;
};
if(state.original){mkdirSync(path.dirname(backup),{recursive:true});execute('ditto',[bundle,backup]);state.originalInventory=inventory(bundle);if(JSON.stringify(inventory(backup))!==JSON.stringify(state.originalInventory))throw new Error('Backup mismatch');}
save();
function request(message){return new Promise((resolve,reject)=>{const socket=net.createConnection(state.fixtureSocket);let data='';socket.setTimeout(3000,()=>socket.destroy(new Error('fixture timeout')));socket.on('connect',()=>socket.write(JSON.stringify(message)+'\n'));socket.on('data',chunk=>{data+=chunk;if(data.includes('\n')){socket.end();try{resolve(JSON.parse(data.slice(0,data.indexOf('\n'))));}catch(e){reject(e);}}});socket.on('error',reject);});}
async function stop(child){if(!child||child.exitCode!==null)return;child.kill('SIGTERM');await Promise.race([new Promise(resolve=>child.once('exit',resolve)),new Promise(resolve=>setTimeout(resolve,2000))]);if(child.exitCode===null)child.kill('SIGKILL');}
async function restore(){if(finished)return;state.phase='restoring';save();const errors=[];try{execute(sources,['select',state.baseline.current]);}catch(e){errors.push('source selection: '+e.message);}await stop(fixture);await stop(host);try{const disabled=JSON.parse(execute(path.join(base,'source-registration'),[bundle,'disable']));writeFileSync(path.join(run,'disabled.json'),JSON.stringify(disabled,null,2)+'\n');const parent=disabled.sources.find(source=>source.id==='si.astrafoundry.inputmethod.arion');if(disabled.sources.length&&(!parent||parent.enabled||disabled.sources.some(source=>source.selected)||Object.values(disabled.statuses).some(status=>status!==0)))throw new Error('Diagnostic parent or selection remains active');rmSync(bundle,{recursive:true,force:true});}catch(e){errors.push('remove: '+e.message);}try{if(state.original){execute('ditto',[backup,bundle]);if(JSON.stringify(inventory(bundle))!==JSON.stringify(state.originalInventory))throw new Error('Restored byte inventory mismatch');}}catch(e){errors.push('bundle restore: '+e.message);}try{execute(sources,['select',state.baseline.current]);state.after=JSON.parse(execute(sources,[]));}catch(e){errors.push('source recheck: '+e.message);}state.phase=errors.length?'restore-failed':'restored';state.errors=errors;save();finished=true;return state;}
for(const signal of ['SIGINT','SIGTERM','SIGHUP'])process.on(signal,()=>restore().finally(()=>process.exit(1)));
process.on('uncaughtException',error=>{process.stderr.write(String(error)+'\n');restore().finally(()=>process.exit(1));});
process.stdout.write(JSON.stringify({ready:true,...state})+'\n');
for await(const line of createInterface({input:process.stdin})){
 try{
 const command=JSON.parse(line);let result;
 if(command.op==='install'){
  execute('codesign',['--verify','--deep','--strict',staged]);
  execute('ditto',[staged,bundle]);result=JSON.parse(execute(path.join(base,'source-registration'),[bundle,'enable']));writeFileSync(path.join(run,'registration.json'),JSON.stringify(result,null,2)+'\n');state.phase='installed';save();
 }
 else if(command.op==='fixture'){result=await request(command.command);}
 else if(command.op==='sources'){result=JSON.parse(execute(sources,[]));}
 else if(command.op==='finish'){result=await restore();process.stdout.write(JSON.stringify({op:command.op,result})+'\n');break;}
 else throw new Error('unknown controller operation');
 process.stdout.write(JSON.stringify({op:command.op,result})+'\n');
 }catch(error){process.stdout.write(JSON.stringify({error:String(error),stdout:error.stdout?.toString(),stderr:error.stderr?.toString()})+'\n');}
}
if(!finished)await restore();
