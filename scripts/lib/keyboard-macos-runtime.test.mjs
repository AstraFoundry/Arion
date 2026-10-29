import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdirSync,mkdtempSync,writeFileSync,rmSync,symlinkSync} from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import {verifyMacRuntimeCache} from './keyboard-macos-runtime.mjs';

test('cache requires both exact artifacts and rejects corruption or redirected output',()=>{
 const directory=mkdtempSync(path.join(os.tmpdir(),'arion-runtime-cache-'));
 const identity={source:'reviewed'},version='1.17.0';
 const library=`build/lib/librime.${version}.dylib`,deployer='build/bin/rime_deployer';
 const bytes=Buffer.from('owned cache fixture');
 const item={sizeBytes:bytes.length,sha256:createHash('sha256').update(bytes).digest('hex')};
 const record={identity,library,deployer,outputs:{[library]:item,[deployer]:item}};
 const verify=changed=>verifyMacRuntimeCache({directory,identity,version,record:changed});
 try {
  for(const file of [library,deployer]) {
   mkdirSync(path.dirname(path.join(directory,file)),{recursive:true});
   writeFileSync(path.join(directory,file),bytes);
  }
  assert.doesNotThrow(()=>verify(record));
  for(const changed of [
   {...record,identity:{source:'other'}},
   {...record,outputs:{}},
   {...record,outputs:{[library]:item}},
   {...record,library:'../unchecked.dylib'},
   {...record,outputs:{...record.outputs,extra:item}},
   {...record,outputs:{...record.outputs,[library]:{...item,sha256:'0'.repeat(64)}}},
  ]) assert.throws(()=>verify(changed),/cache/);
  writeFileSync(path.join(directory,library),Buffer.alloc(bytes.length));
  assert.throws(()=>verify(record),/Corrupt/);
  rmSync(path.join(directory,library));
  symlinkSync(path.join(directory,deployer),path.join(directory,library));
  assert.throws(()=>verify(record),/Corrupt/);
 } finally {rmSync(directory,{recursive:true,force:true});}
});
