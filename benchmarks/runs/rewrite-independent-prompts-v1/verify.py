"""Verify independent prompt authorship and empty-example packaging offline."""
import hashlib,json,pathlib,shlex
ROOT=pathlib.Path(__file__).resolve().parents[3]
RUN=ROOT/'benchmarks/runs/rewrite-independent-prompts-v1'
HERE=pathlib.Path(__file__).resolve().parent

def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def answer(name):
 events=[json.loads(z) for z in (RUN/name).read_text().splitlines()]
 if any(e['type'] in ['tool_use','error'] for e in events):raise ValueError('Unexpected tool/error in accepted author response')
 return json.loads(''.join(e.get('part',{}).get('text','') for e in events if e['type']=='text'))
def verify():
 archive=json.loads((RUN/'archive.json').read_text())
 for path,digest in archive['files'].items():
  if sha(RUN/path)!=digest:raise ValueError('Changed authoring archive: '+path)
 manifest=json.loads((RUN/'isolated-authoring-manifest.json').read_text());session=json.loads((RUN/'isolated-session-export.json').read_text());info=session['info']
 if info['id']!=manifest['sessionID'] or info['location']['directory']!=manifest['explicitLocation'] or '/code/Arion' in info['location']['directory']:raise ValueError('Wrong authoring session location')
 if info['model']['id']!='gemini-4-argon-contributor' or info['model']['providerID']!='opencode-go':raise ValueError('Wrong author model')
 if info['permissions']!=[{'action':'*','resource':'*','effect':'deny'}]:raise ValueError('Author tools not denied')
 users=[m for m in session['messages'] if 'text' in m and 'files' in m]
 if len(users)!=2 or any(u['files'] for u in users):raise ValueError('Unexpected supplied author inputs')
 # The installed CLI records its single message argument with shell quoting.
 for user,name in zip(users,['isolated-author-brief.txt','output-contract-brief.txt']):
  if shlex.split(user['text'])!=[(RUN/name).read_text()]:raise ValueError('Unrecorded author input')
 initial=answer('initial-isolated-events.jsonl');final=answer('isolated-events.jsonl')
 if initial['selection']!=final['selection']:raise ValueError('Unexpected selector revision')
 for k in initial['format']:
  if k not in ['formatSystem','formatSystemHan'] and initial['format'][k]!=final['format'][k]:raise ValueError('Unexpected formatter policy revision')
 if json.loads((HERE/'candidate.json').read_text())!=final:raise ValueError('Candidate was manually rewritten')
 for name in ['selection','format']:
  if json.loads((HERE/(name+'.json')).read_text())!=final[name]:raise ValueError('Split prompt differs from model response')
 if final['selection']['selectionExamples'] or any(v['examples'] for v in final['format']['format'].values()):raise ValueError('Demonstrations present')
 print(json.dumps(dict(verified=True,model='opencode-go/gemini-4-argon-contributor',acceptedAuthorTurns=2,toolCalls=0,examples=0,localQwenEvaluationRuns=0,productionActivated=False)))
if __name__=='__main__':verify()
