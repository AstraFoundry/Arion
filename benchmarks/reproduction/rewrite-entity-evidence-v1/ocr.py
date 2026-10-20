"""Run the current production Vision recognizer on pinned authored images."""
import argparse
import json
import os
import pathlib
import platform
import subprocess
import run as study
HERE=study.HERE
ROOT=study.ROOT
engine=study.engine
BINARY=ROOT/'.workspace/rewrite-entity-evidence-v1/arion-ocr'
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();study.check_freeze()
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=dict(platform=platform.platform(),machine=platform.machine(),goVersion=subprocess.check_output(['go','version'],text=True).strip(),binarySha256=engine.sha(BINARY),freezeSha256=engine.sha(HERE/'freeze.json'))
    manifest=dict(status='running',provenance=provenance);engine.write(out/'manifest.json',manifest)
    with (out/'stderr.log').open('x') as stream:
        result=subprocess.run([str(BINARY),'-corpus',str(study.CORPUS/'corpus.json'),'-out',str(out/'observations.jsonl')],stderr=stream)
    if result.returncode:
        manifest.update(status='failed',exitCode=result.returncode);engine.write(out/'manifest.json',manifest);raise RuntimeError('Native OCR failed')
    samples=study.samples_for('ocr',out/'observations.jsonl')
    manifest.update(status='complete',observations=len(samples),observationsSha256=engine.sha(out/'observations.jsonl'));engine.write(out/'manifest.json',manifest);print(json.dumps(manifest,indent=2))
if __name__=='__main__':main()
