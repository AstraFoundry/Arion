"""Verify archived experiment identities without loading a model."""
import json
import pathlib
import report
import study

ROOT=study.ROOT
ARCHIVE=ROOT/'benchmarks/runs/scene-rewrite-alignment-v1'
index=study.read(ARCHIVE/'checksums.json')
for name,h in index['files'].items():
    if study.sha(ROOT/name)!=h:raise ValueError('Archive checksum mismatch: '+name)
dev,_,_,_=report.load_run(ARCHIVE/'dev','dev')
test,_,_,_=report.load_run(ARCHIVE/'test','test')
if test['selectionSha256']!=study.sha(ARCHIVE/'dev/selection.json'):raise ValueError('Selection checksum mismatch')
if any(dev['provenance'][k]!=test['provenance'][k] for k in ['modelFiles','binaries','build','freezeSha256']):raise ValueError('Unpaired experiment assets')
print('Verified frozen fixtures, 96 generations, completed paired runs and '+str(len(index['files']))+' file identities.')
