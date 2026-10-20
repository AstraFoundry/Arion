"""Replay the archived production cleanup implementation in an isolated Go module."""
import argparse,json,pathlib,shutil,subprocess
import verify
HERE=verify.HERE
parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();verify.checked();out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
shutil.copytree(HERE/'sources/domain',out/'domain');(out/'go.mod').write_text('module arion.local/core\n\ngo 1.25.7\n');shutil.copyfile(HERE/'main.go',out/'main.go')
import os
result=subprocess.run(['go','run','.',str(HERE/'inputs.json')],cwd=out,env=dict(os.environ,GOWORK='off'),stdout=subprocess.PIPE,stderr=subprocess.PIPE,check=True)
(out/'observations.json').write_bytes(result.stdout)
if result.stdout!=(verify.RUN/'observations.json').read_bytes():raise ValueError('Cleanup replay differs from recorded production source')
print('Byte-identical archived CleanSpeech replay; no LLM inference')
