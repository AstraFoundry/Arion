import {execFileSync,spawnSync} from 'node:child_process';
import {existsSync,mkdirSync,readFileSync,writeFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import path from 'node:path';
import {macosSigningSettings,prepareMacOSSigning,stripAppleCredentials} from '../../../scripts/lib/desktop-signing.mjs';
const root=process.cwd(), run=path.join(root,'.workspace/e2e/imk-system/run-13');
if(existsSync(run))throw new Error('Existing run must not be overwritten');
mkdirSync(run,{recursive:true});
if(existsSync('.env'))process.loadEnvFile('.env');
const settings=macosSigningSettings(process.env);if(!settings)throw new Error('Configured signing required');
const signing=prepareMacOSSigning(settings);
const env=stripAppleCredentials(process.env);delete env.MYGO_UPDATER_PRIVATE_KEY;
const exec=(command,args)=>execFileSync(command,args,{cwd:root,env,encoding:'utf8',stdio:['ignore','pipe','pipe']});
const archive=path.join(root,'.workspace/research/imk-alignment/registration-audit/metadata-sources/Apple-NumberInput.zip');
const hash=file=>createHash('sha256').update(readFileSync(file)).digest('hex');
if(hash(archive)!=='06b6f0052b7bb5bc584fb6d32a9d6fda4bba1746d29f56f00123466b60729228')throw new Error('Apple sample source changed');
const bundle=path.join(run,'payload/NumberInputControl.app'), src=path.join(run,'source');
const parentID='si.astrafoundry.inputmethod.numbercontrol', connection='ArionNumberControl_Connection';
mkdirSync(src);mkdirSync(path.join(bundle,'Contents/MacOS'),{recursive:true});
const resources=path.join(bundle,'Contents/Resources');mkdirSync(resources,{recursive:true});
const prefix='NumberInput_IMKit_Sample/trunk/NumberInput 0/';
const members=['main.m','NumberInputController.h','NumberInputController.m','Info.plist','English.lproj/InfoPlist.strings','English.lproj/MainMenu.nib/classes.nib','English.lproj/MainMenu.nib/info.nib','English.lproj/MainMenu.nib/keyedobjects.nib','nine.tiff'];
const provenance=[];
try {
 for(const member of members){
  const bytes=execFileSync('unzip',['-p',archive,prefix+member],{env});
  const original=path.join(src,member);mkdirSync(path.dirname(original),{recursive:true});writeFileSync(original,bytes);
  provenance.push({member,sha256:hash(original)});
  if(!member.endsWith('.m')&&!member.endsWith('.h')&&member!=='Info.plist'){
   const destination=path.join(resources,member);mkdirSync(path.dirname(destination),{recursive:true});writeFileSync(destination,bytes);
  }
 }
 const main=readFileSync(path.join(src,'main.m'),'utf8').replaceAll('NumberInput_1_Connection',connection);
 writeFileSync(path.join(src,'main-derived.m'),main);
 const info=JSON.parse(exec('plutil',['-convert','json','-o','-',path.join(src,'Info.plist')]));
 info.CFBundleIdentifier=parentID;info.CFBundleName='Arion Number Control';info.InputMethodConnectionName=connection;
 info.LSMinimumSystemVersion='14.0';info.CFBundleShortVersionString='1.0';
 const metadata=path.join(bundle,'Contents/Info.plist');writeFileSync(metadata,JSON.stringify(info));exec('plutil',['-convert','xml1',metadata]);
 const binary=path.join(bundle,'Contents/MacOS/NumberInput');
 exec('xcrun',['clang','-target','arm64-apple-macos14.0','-O2','-I',src,path.join(src,'main-derived.m'),path.join(src,'NumberInputController.m'),'-framework','Cocoa','-framework','InputMethodKit','-o',binary]);
 exec('codesign',['--force','--options','runtime','--timestamp','--sign',settings.identity,bundle]);
 exec('codesign',['--verify','--deep','--strict',bundle]);
 writeFileSync(path.join(run,'artifact.json'),JSON.stringify({purpose:'Current-SDK official Apple stage-0 IMK registration control, not Arion typing acceptance',parentID,connection,architecture:'arm64',sampleURL:'https://developer.apple.com/library/archive/samplecode/NumberInput_IMKit_Sample/NumberInput_IMKit_Sample.zip',sampleSHA256:hash(archive),provenance,changes:['bundle ID/name','unique connection in plist and main','minimum OS and marketing version'],binarySHA256:hash(binary),infoSHA256:hash(metadata)},null,2)+'\n');
 const zip=path.join(run,'NumberInputControl.zip');exec('ditto',['-c','-k','--keepParent',bundle,zip]);
 const args=['notarytool','submit',zip,'--keychain-profile',settings.profile,'--wait','--timeout','10m','--output-format','json'];if(settings.keychain)args.push('--keychain',settings.keychain);
 const submission=JSON.parse(exec('xcrun',args));writeFileSync(path.join(run,'notarization.json'),JSON.stringify(submission,null,2)+'\n');if(submission.status!=='Accepted')throw new Error('Control notarization rejected');
 exec('xcrun',['stapler','staple',bundle]);
 const assessment=spawnSync('spctl',['--assess','--type','execute','--verbose=2',bundle],{env,encoding:'utf8'});
 writeFileSync(path.join(run,'gatekeeper.json'),JSON.stringify({status:assessment.status,stdout:assessment.stdout,stderr:assessment.stderr},null,2)+'\n');if(assessment.status!==0)throw new Error('Control Gatekeeper rejected');
 console.log(JSON.stringify({prepared:true,parentID,notarization:submission.status}));
}finally{signing.cleanup();}
