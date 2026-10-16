import hashlib,json
from pathlib import Path
root=Path(__file__).resolve().parent
corpus=json.loads((root/'corpus.json').read_text()); by={s['id']:s for s in corpus['samples']}
def score(path):
 rows=[json.loads(l) for l in path.read_text().splitlines()]
 if len(rows)!=len(by) or len({r['caseId'] for r in rows})!=len(by):raise ValueError('Incomplete fixture coverage')
 results={}; languages={}; rejects={}
 for r in rows:
  s=by[r['caseId']]
  good=lambda text:all(t in text for t in s['requiredStrings']+s['protected']) and not any(t in text for t in s['forbiddenStrings'])
  passed=good(r['text']) and not r['truncated']
  results[s['id']]=passed
  entry=languages.setdefault(s['language'],dict(cases=0,deliveredPass=0,proposalPass=0,rejectedPassing=0))
  entry['cases']+=1; entry['deliveredPass']+=int(passed);entry['proposalPass']+=int(good(r['proposal']))
  entry['rejectedPassing']+=int(good(r['proposal']) and r['verdict']!='accepted')
  rejects[r['verdict']]=rejects.get(r['verdict'],0)+1
 return dict(cases=len(rows),byLanguage=languages,verdicts=rejects,deliveredPass=sum(results.values())),results
out={'schemaVersion':1,'kind':'unreviewed-multilingual-text-probe','languages':{'productRegression':['zh','en'],'exploratoryTextOnly':['es','fr','ja']},'prompt':'live-v2','runs':{},'limits':['Developer-authored synthetic text, no independent meaning review','No ASR/audio or native target insertion measured','Foreign-language text probes do not establish recognition language support','4B is an exploratory model comparison, not an adopted default','No timing or throughput claim; foreground processes/power not isolated']}
results={}
for name in ['baseline','baseline-4b','final']:
 summary,passed=score(root/(name+'.jsonl'));out['runs'][name]=summary;results[name]=passed
out['paired']={'improved':sum(not results['baseline'][i] and results['final'][i] for i in by),'regressed':sum(results['baseline'][i] and not results['final'][i] for i in by)}
out['hashes']={name:hashlib.sha256((root/name).read_bytes()).hexdigest() for name in ['corpus.json','baseline.jsonl','baseline-4b.jsonl','final.jsonl']}
(root/'summary.json').write_text(json.dumps(out,indent=2)+'\n')
print(json.dumps({'runs':out['runs'],'paired':out['paired']}))
