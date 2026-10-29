"""Verify every retained byte and independently replay all report fields."""
import argparse,hashlib,importlib.util,json,pathlib,tarfile,tempfile,shutil
from report import evaluate
p=argparse.ArgumentParser();p.add_argument('archive');p.add_argument('--negative-tests',action='store_true');a=p.parse_args()
directory=pathlib.Path(a.archive).resolve();manifest=json.loads((directory/'archive.json').read_text());packed=directory/manifest['archive']
assert hashlib.sha256(packed.read_bytes()).hexdigest()==manifest['archiveSha256']
with tempfile.TemporaryDirectory(prefix='arion-ime-replay-') as temporary:
 root=pathlib.Path(temporary)
 with tarfile.open(packed,'r:gz') as tar:
  names={m.name for m in tar.getmembers()};assert len(names)==len(manifest['entries'])
  for entry in manifest['entries']:
   member=tar.getmember(entry['member']);assert member.isfile() and not pathlib.PurePosixPath(member.name).is_absolute() and '..' not in pathlib.PurePosixPath(member.name).parts
   data=tar.extractfile(member).read();assert len(data)==entry['sizeBytes'] and hashlib.sha256(data).hexdigest()==entry['sha256']
   file=root/member.name;file.parent.mkdir(parents=True,exist_ok=True);file.write_bytes(data)
 run=root/'evidence/run-v1';replayed=evaluate(run);assert replayed==json.loads((directory/'report.json').read_text())
 rejected=[]
 if a.negative_tests:
  def rejection(name,modify,restore):
   modify()
   try:evaluate(run)
   except (AssertionError,KeyError,ValueError):rejected.append(name)
   else:raise AssertionError('Invalid evidence accepted: '+name)
   finally:restore()
  status=run/'status.json';original=status.read_bytes()
  rejection('incomplete run',lambda:status.write_text('{"status":"running"}'),lambda:status.write_bytes(original))
  index_file=run/'index.json';index_bytes=index_file.read_bytes();index=json.loads(index_bytes);raw=run/index[0]['path'];raw_bytes=raw.read_bytes()
  rejection('changed raw bytes',lambda:raw.write_bytes(raw_bytes+b' '),lambda:raw.write_bytes(raw_bytes))
  duplicate=[index[0],index[0]]+index[2:]
  rejection('duplicate observation',lambda:index_file.write_text(json.dumps(duplicate)),lambda:index_file.write_bytes(index_bytes))
  record=json.loads(raw_bytes);request=run/record['request'];request_bytes=request.read_bytes();leaked=json.loads(request_bytes);leaked['target']='reference-leak'
  rejection('request reference leakage',lambda:request.write_text(json.dumps(leaked)),lambda:request.write_bytes(request_bytes))
 print(json.dumps({'status':'passed','originalFiles':len(manifest['entries']),'nativeObservations':replayed['nativeObservations'],'exactReportReplay':True,'rejected':rejected},indent=2))
