"""Build isolated research overlays; production source files are never edited."""
import hashlib
import json
import pathlib
import subprocess

HERE = pathlib.Path(__file__).resolve().parent
ROOT = next(p for p in HERE.parents if (p/'models/catalog.json').exists())
OUT = ROOT/'.workspace/scene-rewrite-alignment-v1/build'
OUT.mkdir(parents=True, exist_ok=False)
worker = ROOT/'apps/desktop/cmd/arion-llm/main.go'
prompt = ROOT/'apps/desktop/infrastructure/llm/prompt/prompt.go'
engine = ROOT/'apps/desktop/infrastructure/llm/engine/engine.go'
w = worker.read_text()
w = w.replace('template := prompt.Live()', '''template := prompt.Live()
	contextual := prompt.Context()
	if path := os.Getenv("ARION_EXPERIMENT_TEMPLATE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil { return fail(err) }
		if err := json.Unmarshal(data, &template); err != nil { return fail(err) }
		contextual = template
	}''')
w = w.replace('}, template, prompt.Context())', '}, template, contextual)')
assert 'ARION_EXPERIMENT_TEMPLATE' in w and '}, template, contextual)' in w
p = prompt.read_text().replace('range domain.ModelScenes', 'range domain.Scenes')
# Close the dictation section consistently for examples and final requests.
p = p.replace('b.WriteString(in.Text)\n\treturn b.String()', 'b.WriteString(in.Text)\n\tif !in.Context.Bounded().Empty() { b.WriteString("\\n</dictation>") }\n\treturn b.String()')
assert '\\n</dictation>' in p
(OUT/'worker.go').write_text(w)
(OUT/'prompt.go').write_text(p)
e = engine.read_text().replace('"errors"', '"errors"\n "os"')
e = e.replace('cleanup := domain.CleanSpeech(req.Text)', 'cleanup := domain.CleanSpeech(req.Text)\n if os.Getenv("ARION_EXPERIMENT_AUDIT") == "1" { cleanup = domain.SpeechCleanup{Text: req.Text} }')
(OUT/'engine.go').write_text(e)
overlay = {'Replace': {str(worker): str(OUT/'worker.go'), str(prompt): str(OUT/'prompt.go'), str(engine): str(OUT/'engine.go')}}
(OUT/'overlay.json').write_text(json.dumps(overlay, indent=2)+'\n')
import os
env = dict(os.environ, CGO_ENABLED='0')
subprocess.run(['go','build','-trimpath','-overlay',str(OUT/'overlay.json'),'-o',str(OUT/'worker'),'./cmd/arion-llm'],cwd=ROOT/'apps/desktop',env=env,check=True)
subprocess.run(['go','build','-trimpath','-o',str(OUT/'validate'),'./cmd/arion-context-probe'],cwd=ROOT/'apps/desktop',env=env,check=True)
files = [worker,prompt,engine,OUT/'worker.go',OUT/'prompt.go',OUT/'engine.go',OUT/'worker',OUT/'validate',HERE/'build.py']
(OUT/'manifest.json').write_text(json.dumps({str(p.relative_to(ROOT)):hashlib.sha256(p.read_bytes()).hexdigest() for p in files},indent=2)+'\n')
print(OUT)
