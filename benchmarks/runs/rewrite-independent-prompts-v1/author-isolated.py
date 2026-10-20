import json,pathlib,subprocess,hashlib
w=pathlib.Path('/Users/steven/code/Arion/.workspace/rewrite-independent-prompts-v1');session=json.load(open(w/'explicit-session-create.json'))['data'];cwd=session['location']['directory'];model='opencode-go/gemini-4-argon-contributor'
assert 'arion-clean-prompt-' in cwd and '/code/Arion' not in cwd
brief=(w/'author-brief.txt').read_text()
brief=brief.replace('Keep when evidence is insufficient or conflicting, not simply because the original reads fluently.','Keep when evidence is insufficient or conflicting, not simply because the original reads fluently.')
brief += '''\nGeneral consistency requirements: Grammatical fluency or a plausible original word alone cannot veto a context-supported spelling correction that preserves referent and meaning; ASR errors can read fluently. Concision must never mean summarizing away meaningful information. Do not claim that prompt/input/output token budgets fit without actual measurements; capacity depends on input length and number of hypotheses.\n'''
(w/'isolated-author-brief.txt').write_text(brief)
manifest=dict(requestedModel=model,sessionID=session['id'],explicitLocation=cwd,permissions=session['permissions'],briefSha256=hashlib.sha256(brief.encode()).hexdigest(),providedArtifacts=['isolated-author-brief.txt'],excludedInputs=['prior model drafts','product prompt files and examples','test cases, reference answers and results','Arion repository or task conversation'],status='running')
(w/'isolated-authoring-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
cmd=['/Users/steven/.opencode/bin/opencode','run','--session',session['id'],'--model',model,'--format','json',brief]
with (w/'isolated-events.jsonl').open('w') as out,(w/'isolated-stderr.log').open('w') as error:r=subprocess.run(cmd,cwd=cwd,stdout=out,stderr=error)
manifest.update(status='complete' if r.returncode==0 else 'failed',exitCode=r.returncode);(w/'isolated-authoring-manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
export=subprocess.run(['/Users/steven/.opencode/bin/opencode','session','export',session['id']],cwd=cwd,capture_output=True,text=True);(w/'isolated-session-export.json').write_text(export.stdout)
print('author exit',r.returncode,'export exit',export.returncode)
