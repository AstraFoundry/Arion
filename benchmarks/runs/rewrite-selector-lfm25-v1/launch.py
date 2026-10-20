"""Require terminal native qualification and frozen supplemental sources before comparison."""
from pathlib import Path
import subprocess,json,hashlib
R=Path.cwd();W=Path(__file__).resolve().parent;H=R/'benchmarks/reproduction/rewrite-selector-lfm25-v1'
assert json.loads((W/'preflight-terminal-status.json').read_text())['exitCode']==0
assert json.loads((W/'qualification-status.json').read_text())['checked'] is True
for path,digest in json.loads((H/'comparison-sources.json').read_text())['files'].items():
 assert hashlib.sha256((R/path).read_bytes()).hexdigest()==digest,path
cmd=['python3',str(H/'run.py'),'--out',str(W/'run')]
with (W/'execution.log').open('x') as f:code=subprocess.call(cmd,stdout=f,stderr=subprocess.STDOUT)
(W/'terminal-status.json').write_text(json.dumps(dict(command=cmd,exitCode=code,comparisonSourcesSha256=hashlib.sha256((H/'comparison-sources.json').read_bytes()).hexdigest()))+'\n');print('LFM2.5202 comparison exit',code);raise SystemExit(code)
