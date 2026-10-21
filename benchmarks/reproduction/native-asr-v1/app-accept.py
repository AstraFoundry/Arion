from pathlib import Path
import json,os,subprocess,socket,time
root=Path.cwd();work=root/'.workspace/native-asr-migration';data=work/'app-offline-profile';data.mkdir(exist_ok=True)
if not (data/'models').exists():os.symlink(work/'model-store',data/'models')
control=work/'app-offline.sock';control.unlink(missing_ok=True);runtime=Path('/tmp')/('arion-native-accept-'+str(os.getuid()));runtime.mkdir(mode=0o700,exist_ok=True)
exe=root/'apps/desktop/build/darwin-arm64/Arion.app/Contents/MacOS/Arion'
env=os.environ.copy();env.update(ARION_DATA_DIR=str(data),ARION_OFFLINE='1',ARION_E2E_CONTROL=str(control),XDG_RUNTIME_DIR=str(runtime));env.pop('ARION_ASR_WORKER',None)
log=(work/'app-offline-isolated.log').open('w');p=subprocess.Popen([str(exe)],env=env,stdout=log,stderr=log)
try:
 deadline=time.monotonic()+90
 while not control.exists():
  if p.poll() is not None:raise RuntimeError('app exited before control socket')
  if time.monotonic()>deadline:raise TimeoutError('control socket')
  time.sleep(.2)
 client=socket.socket(socket.AF_UNIX,socket.SOCK_STREAM);client.connect(str(control));stream=client.makefile('rwb',buffering=0)
 while True:
  stream.write(b'{"op":"state"}\n');reply=json.loads(stream.readline());state=reply.get('observations',reply)
  if state.get('model',{}).get('state')=='ready':break
  if time.monotonic()>deadline:raise TimeoutError(str(reply))
  time.sleep(.5)
 (work/'app-offline-acceptance.json').write_text(json.dumps({'networkMode':'ARION_OFFLINE=1','isolatedRuntimeDirectory':True,'workerOverride':False,'response':reply},indent=2)+'\n')
 print('Packaged bootstrap model state:',state.get('model'))
 stream.write(b'{"op":"quit"}\n');p.wait(timeout=20);print('App exit:',p.returncode)
finally:
 if p.poll() is None:p.terminate();p.wait(timeout=10)
 try:runtime.rmdir()
 except OSError:pass
