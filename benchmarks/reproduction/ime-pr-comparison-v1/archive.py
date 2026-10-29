"""Preserve original evidence bytes in a compact, model-free replayable archive."""
import argparse,hashlib,json,pathlib,tarfile,shutil
from report import evaluate
from run import load_variants
p=argparse.ArgumentParser();p.add_argument('work');p.add_argument('destination');p.add_argument('--variants');a=p.parse_args()
variants=load_variants(a.variants)
work=pathlib.Path(a.work).resolve();destination=pathlib.Path(a.destination).resolve();destination.mkdir(parents=True,exist_ok=False)
assert json.loads((work/'run-v1/status.json').read_text())['status']=='completed'
entries=[]
def add(tar,file,member):
 if not file.is_file() or file.is_symlink():return
 data=file.read_bytes();entries.append({'original':str(file),'member':member,'sizeBytes':len(data),'sha256':hashlib.sha256(data).hexdigest()});tar.add(file,arcname=member,recursive=False)
with tarfile.open(destination/'evidence.tar.gz','w:gz',compresslevel=6) as tar:
 for directory in ['run-v1','smoke-v2','profiles-v1','defaults-v1','folded-mixed-v1','native-negative-v1','validation','attempts','apostrophe-check']:
  for file in sorted((work/directory).rglob('*')):add(tar,file,'evidence/'+str(file.relative_to(work)))
 for name in ['versions.json','claim.md','pr40-stage.stdout','pr40-stage.stderr','pr45-stage.stdout','pr45-stage.stderr','build40.stdout','build40.stderr','build45.stdout','build45.stderr','smoke-v1.stdout','smoke-v1.stderr','smoke-v2.stdout','smoke-v2.stderr','run-v1.stdout','run-v1.stderr','profiles-v1.stdout','profiles-v1.stderr','defaults-v1.stdout','defaults-v1.stderr','folded-mixed-v1.stdout','folded-mixed-v1.stderr']:
  add(tar,work/name,'evidence/'+name)
 for name in ['variants.json','product-source.patch','build-integrated.stdout','build-integrated.stderr']:
  add(tar,work/name,'evidence/'+name)
 for variant,config in variants.items():
  for prefix in ['apps/desktop/ime','scripts/lib']:
   for file in sorted((work/config['snapshot']/prefix).rglob('*')):add(tar,file,'sources/'+variant+'/'+str(file.relative_to(work/config['snapshot'])))
  for relative in ['scripts/keyboard-runtime.json','scripts/stage-keyboard-runtime.mjs','apps/desktop/desktop.json','apps/desktop/resources/darwin/keyboard/runtime.json']:
   add(tar,work/config['snapshot']/relative,'sources/'+variant+'/'+relative)
 for file in sorted((work/list(variants.values())[1]['snapshot']/'.workspace/keyboard-runtime/source-build').rglob('*')):
  if '/evidence/' in str(file) or file.name=='build.json':add(tar,file,'runtime-build/'+str(file.relative_to(work/list(variants.values())[1]['snapshot']/'.workspace/keyboard-runtime/source-build')))
 for prefix in ['benchmarks/reproduction/ime-pr-comparison-v1','benchmarks/corpora/ime-pr-comparison-v1','benchmarks/corpora/ime-spelling-reference-v1']:
  for file in sorted((pathlib.Path(__file__).resolve().parents[3]/prefix).iterdir()):
   if file.suffix in ['.py','.mm','.mjs','.md','.json','.cc']:add(tar,file,'study-source/'+prefix+'/'+file.name)
 for build in [v['build'] for v in variants.values()]:
  add(tar,work/build/'identity.json','evidence/'+build+'/identity.json')
archive=destination/'evidence.tar.gz'
(destination/'archive.json').write_text(json.dumps({'format':'original-byte-tar-gzip-v1','archive':archive.name,'archiveSha256':hashlib.sha256(archive.read_bytes()).hexdigest(),'entries':entries,'excluded':'Native binaries, runtime libraries/compiled data, SDKs, build caches, profiles and keys. Their source/artifact identities remain in evidence.'},indent=2)+'\n')
report=evaluate(work/'run-v1');(destination/'report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print('Archived',len(entries),'original files;',archive.stat().st_size,'compressed bytes')
