"""Replay native observations with evaluator-only labels and hash checks."""
import argparse,collections,hashlib,json,pathlib,statistics,re
ROOT=pathlib.Path(__file__).resolve().parents[3]
def read(path):return json.loads(pathlib.Path(path).read_text())
def hash_file(path):return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
def percentile(values,q):
 a=sorted(values);p=(len(a)-1)*q;lo=int(p);hi=min(lo+1,len(a)-1)
 return a[lo]+(a[hi]-a[lo])*(p-lo)
def distribution(values):
 return None if not values else {'count':len(values),'p50':percentile(values,.5),'p95':percentile(values,.95),'p99':percentile(values,.99),'max':max(values)}
def metrics(row,fixture):
 state=row['initial']['state'];target=fixture.get('target');candidates=state['candidates'];final=row['final']
 m={'firstPage':candidates,'allKeysHandled':all(s['handled'] for s in row['initial']['steps']),'keyMs':[s['nativeStateMs'] for s in row['initial']['steps']],
    'startupMs':row['initial']['startupMs'],'typingNativeMs':sum(s['nativeStateMs'] for s in row['initial']['steps']),
    'selectionMs':[s['nativeStateMs'] for s in row['actions'] if s['op']=='select'],'pageMs':[s['nativeStateMs'] for s in row['actions'] if s['op']=='next'],
    'peakRSSBytes':row['terminal']['peakRSSBytes'],'finalText':final['text'],'insertions':final['insertions']}
 if target:
  exact=next((i+1 for i,c in enumerate(candidates) if c==target),None)
  folded=next((i+1 for i,c in enumerate(candidates) if c.lower()==target.lower()),None)
  m.update(rank=exact,caseInsensitiveRank=folded,lookupRank=row['searches'][0].get('rank'),
     commitExact=final['text']==target and final['insertions']==1 and not final['marked'] and not final['composing'])
 elif fixture['group']=='mixed':
  expected=''.join(fixture['segments'])
  m.update(segmentRanks=[s.get('rank') for s in row['searches']],commitExact=final['text']==expected and final['insertions']==1 and not final['marked'] and not final['composing'])
 else:
  m.update(englishSlots=sum(bool(re.fullmatch('[A-Za-z]{2,}',c)) for c in candidates),chineseFirst=bool(candidates and re.search('[\u3400-\u9fff]',candidates[0])),
   commitExact=final['insertions']==1 and not final['marked'] and not final['composing'])
 m['success']=m['commitExact'] and m['allKeysHandled']
 return m

def evaluate(run,corpus_file=None):
 run=pathlib.Path(run);corpus_file=pathlib.Path(corpus_file or ROOT/'benchmarks/corpora/ime-pr-comparison-v1/corpus.json')
 corpus=read(corpus_file);fixtures={c['id']:c for c in corpus['cases']};status=read(run/'status.json');assert status['status']=='completed',status
 meta=read(run/'metadata.json');variants=list(meta['versions']);assert hash_file(corpus_file)==meta['corpusSha256'],'Corpus changed'
 index=read(run/'index.json');plan=read(run/'plan.json');assert len(plan)==len(index)==status['observations'];seen=set();rows=[]
 for item,planned in zip(index,plan):
  assert all(item[k]==planned[k] for k in ['variant','round','fixture','pageSize','paceMs'])
  assert item['name'] not in seen;seen.add(item['name'])
  path=run/item['path'];assert hash_file(path)==item['sha256'],str(path)
  row=read(path);assert row['status']=='completed' and row['exitStatus']==0
  fixture=fixtures[row['fixture']];request=read(run/row['request'])
  assert set(request)=={'input','warmups','pageSize','paceMs'} and request['input']==fixture['input']
  assert len(row['initial']['steps'])==len(fixture['input'])
  assert [s['key'] for s in row['initial']['steps']]==[ord(x) for x in fixture['input']]
  for search in row['searches']:
   assert not search['found'] or any(search['target'] in page['candidates'] for page in search['pages'])
  row['metric']=metrics(row,fixture);row['group']=fixture['group'];row['input']=fixture['input'];rows.append(row)
 main=[r for r in rows if r['round']<=5 and r['pageSize']==9 and r['paceMs']==0]
 by_key=collections.defaultdict(list)
 for row in main:by_key[(row['fixture'],row['variant'])].append(row)
 deterministic={}
 for key,repeats in by_key.items():
  signatures=[json.dumps({k:r['metric'].get(k) for k in ['firstPage','finalText','insertions','lookupRank','segmentRanks','success']},ensure_ascii=False,sort_keys=True) for r in repeats]
  deterministic[str(key)]=len(set(signatures))==1
 unique=[v[0] for v in by_key.values()];groups={}
 for variant in variants:
  groups[variant]={}
  for group in sorted({c['group'] for c in corpus['cases']}):
   rr=[r for r in unique if r['variant']==variant and r['group']==group];ms=[r['metric'] for r in rr]
   g={'inputs':len(rr),'successfulExactSingleCommits':sum(m['success'] for m in ms)}
   if group in ['english-exact','english-prefix','explicit-case']:
    g.update(first=sum(m['rank']==1 for m in ms),top5=sum(m['rank'] is not None and m['rank']<=5 for m in ms),top9=sum(m['rank'] is not None for m in ms),top189=sum(m['lookupRank'] is not None for m in ms),foldedFirst=sum(m['caseInsensitiveRank']==1 for m in ms),foldedTop9=sum(m['caseInsensitiveRank'] is not None for m in ms),MRR9=sum(1/m['rank'] if m['rank'] else 0 for m in ms)/len(ms))
   if group=='chinese-control':g.update(quietPages=sum(m['englishSlots']==0 for m in ms),chineseFirst=sum(m['chineseFirst'] for m in ms),totalEnglishSlots=sum(m['englishSlots'] for m in ms))
   groups[variant][group]=g
 paired_success={f for f in fixtures if all(by_key[(f,v)][0]['metric']['success'] for v in variants)}
 speed={}
 for mode,subset in [('accelerated',main),('paced',[r for r in rows if r['paceMs']==80])]:
  speed[mode]={}
  for variant in variants:
   vr=[r for r in subset if r['variant']==variant and r['fixture'] in paired_success];rounds=[]
   for rep in sorted({r['round'] for r in vr}):
    rr=[r for r in vr if r['round']==rep];values=[x for r in rr for x in r['metric']['keyMs']]
    rounds.append({'round':rep,'inputs':len(rr),'keyMs':distribution(values),'startupMs':distribution([r['metric']['startupMs'] for r in rr]),'typingNativeMs':distribution([r['metric']['typingNativeMs'] for r in rr]),'selectMs':distribution([x for r in rr for x in r['metric']['selectionMs']]),'peakRSSMiB':distribution([r['metric']['peakRSSBytes']/1048576 for r in rr])})
   p95=[r['keyMs']['p95'] for r in rounds]
   speed[mode][variant]={'rounds':rounds,'medianRoundP95Ms':statistics.median(p95) if p95 else None,'roundP95RangeMs':[min(p95),max(p95)] if p95 else None,'pairedSuccessfulInputs':len({r['fixture'] for r in vr})}
 five={v:{} for v in variants}
 for v in five:
  for g in groups[v]:
   rr=[r for r in rows if r['pageSize']==5 and r['variant']==v and r['group']==g];ms=[r['metric'] for r in rr]
   five[v][g]={'inputs':len(rr),'top5':sum(m.get('rank') is not None for m in ms),'exactSingleCommits':sum(m['success'] for m in ms),'quietPages':sum(m.get('englishSlots')==0 for m in ms)}
 cases=[]
 for fixture in corpus['cases']:
  cases.append({**fixture,**{v:by_key[(fixture['id'],v)][0]['metric'] for v in variants}})
 followups={}
 for folder in ['defaults-v1','folded-mixed-v1']:
  directory=run.parent/folder
  if not directory.exists():continue
  observations=[]
  for fixture in corpus['cases']:
   if fixture['group']!='mixed':continue
   for variant in variants:
    path=directory/(fixture['id']+'-'+variant+'.json');item=read(path);final=item['final'];expected=''.join(fixture['segments'])
    assert item['terminal']['status']=='completed' and item['input']==fixture['input']
    assert [step['key'] for step in item['initial']['steps']]==[ord(x) for x in fixture['input']]
    same=final['text']==expected if folder=='defaults-v1' else final['text'].lower()==expected.lower()
    success=same and all(step['handled'] for step in item['initial']['steps']) and final['insertions']==1 and not final['marked'] and not final['composing']
    assert success==item['success']
    observations.append({'fixture':fixture['id'],'input':fixture['input'],'variant':variant,'success':success,'text':final['text'],'insertions':final['insertions']})
  followups[folder]={'nativeProcesses':len(observations),'successes':{v:sum(x['success'] for x in observations if x['variant']==v) for v in variants},'observations':observations}
 return {'status':'completed','nativeObservations':len(rows),'uniqueInputs':len(fixtures),'deterministicAllRepeatedOutputs':all(deterministic.values()),'groups':groups,'fiveSlot':five,'speedOnPairedSuccessfulInputs':speed,'cases':cases,'versions':meta['versions'],'boundary':meta['boundary'],'followUps':followups}
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('run');p.add_argument('--corpus');p.add_argument('--output');a=p.parse_args();report=evaluate(a.run,a.corpus);text=json.dumps(report,ensure_ascii=False,indent=2)+'\n'
 if a.output:pathlib.Path(a.output).write_text(text)
 else:print(text)
