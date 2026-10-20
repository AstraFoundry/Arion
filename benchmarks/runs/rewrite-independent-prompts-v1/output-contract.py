import json,pathlib,subprocess
w=pathlib.Path('/Users/steven/code/Arion/.workspace/rewrite-independent-prompts-v1');m=json.load(open(w/'isolated-authoring-manifest.json'));cwd=m['explicitLocation'];sid=m['sessionID']
# Preserve the first correctly isolated author response verbatim.
for name in ['isolated-events.jsonl','isolated-session-export.json']:(w/('initial-'+name)).write_bytes((w/name).read_bytes())
brief='''One interface-contract omission needs correction: the formatter must output only the final edited source body as plain text, with no preface, explanations, JSON envelope, reasoning or surrounding Markdown code fence. Add this output contract to both formatSystem and formatSystemHan. Keep all other prompt strings, scene/route instructions, versions, selection rules and empty example arrays unchanged. Return the full JSON object again. No tools, file reads, test cases, demonstrations or empirical quality claims.'''
(w/'output-contract-brief.txt').write_text(brief)
cmd=['/Users/steven/.opencode/bin/opencode','run','--session',sid,'--model','opencode-go/gemini-4-argon-contributor','--format','json',brief]
with (w/'isolated-events.jsonl').open('w') as out,(w/'output-contract-stderr.log').open('w') as err:r=subprocess.run(cmd,cwd=cwd,stdout=out,stderr=err)
m.update(status='complete' if r.returncode==0 else 'failed',exitCode=r.returncode,revision='Formatter output protocol clarified without examples or evaluation feedback.');(w/'isolated-authoring-manifest.json').write_text(json.dumps(m,indent=2)+'\n')
r2=subprocess.run(['/Users/steven/.opencode/bin/opencode','session','export',sid],cwd=cwd,capture_output=True,text=True);(w/'isolated-session-export.json').write_text(r2.stdout)
print('exit',r.returncode)
