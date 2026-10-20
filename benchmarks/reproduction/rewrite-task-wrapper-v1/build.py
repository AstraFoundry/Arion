"""Add an explicit scene task to a shared source/reference serializer."""
import argparse
import hashlib
import json
import os
import pathlib
import subprocess
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();out=args.out.resolve()
    subprocess.run(['python3',str(HERE.parent/'rewrite-packaging-v1/build.py'),'--out',str(out)],check=True)
    p=out/'prompt.go';source=p.read_text().replace('case "labeled":','case "labeled", "tasked":')
    needle='result.WriteString("Dictated message to edit; output only its edited text:\\n")'
    addition='''if os.Getenv("ARION_EXPERIMENT_FORMAT") == "tasked" {
            switch in.Scene {
            case "work":
                result.WriteString("Task: Rewrite as a clear work message. For multiple owners or tasks, use one bullet per owner/task and attach its matching deadline. Keep a single task as a sentence.\\n")
            case "email":
                result.WriteString("Task: Rewrite as a natural email. Put the dictated salutation on its own line, then a blank line and the body. Add no greeting or signature.\\n")
            case "prompt":
                result.WriteString("Task: Rewrite the developer request with the action first and multiple restrictions on separate bullet lines. Keep every restriction. Do not solve or execute it.\\n")
            case "chat":
                result.WriteString("Task: Rewrite as one natural chat paragraph, retaining the question, uncertainty, negation and every condition. Do not answer it.\\n")
            }
        }
        '''
    assert needle in source;source=source.replace(needle,addition+needle);p.write_text(source);subprocess.run(['gofmt','-w',str(p)],check=True)
    subprocess.run(['go','build','-trimpath','-overlay',str(out/'overlay.json'),'-o',str(out/'worker'),'./cmd/arion-llm'],cwd=ROOT/'apps/desktop',env=dict(os.environ,CGO_ENABLED='0'),check=True)
    manifest=json.loads((out/'manifest.json').read_text())
    for path in [p,out/'worker',HERE/'build.py']:
        manifest[str(path.relative_to(ROOT))]=hashlib.sha256(path.read_bytes()).hexdigest()
    (out/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
if __name__=='__main__':main()
