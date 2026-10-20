"""Record new retrospective review without altering historical inference bytes."""
from pathlib import Path
import hashlib,json
ROOT=Path('.').resolve();OLD=ROOT/'benchmarks/runs/scene-rewrite-alignment-v1';OUT=ROOT/'benchmarks/runs/rewrite-retrospective-v1';OUT.mkdir()
samples={s['id']:s for s in json.loads((ROOT/'benchmarks/reproduction/scene-rewrite-alignment-v1/corpus.json').read_text())['samples']}
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
rh=lambda r:hashlib.sha256(json.dumps(r,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
scene_bad={
'dev-pagination-zh':('Adds completed pagination and implementation explanation, switches language and expands scope.',['invented_completion','answered_request','language_changed']),
'dev-cache-zh':('Implements invented Python cache code instead of rewriting the request; truncated.',['answered_request','truncated']),
'dev-cache-en':('Invents cache-key structure, example data and implementation details; truncated.',['answered_request','truncated']),
'dev-payment-zh':('Invents deployment and extra environment-preparation work; original only says interface finished.',['invented_completion','added_requirement']),
'dev-pagination-en':('Changes requested pagination into completed work and adds review-before-delivery condition from a demonstration.',['invented_completion','added_condition','demonstration_leakage']),
'dev-question-en':('Answers with speculative reasons for slowness instead of preserving the question; truncated.',['answered_question','truncated']),
'dev-owners-en':('Translates the message to Chinese; role/date relationships otherwise preserved.',['language_changed']),
'dev-risk-zh':('Repeats the logging task and adds a separate root-cause-confirmation request.',['added_requirement','repeated_task']),
'dev-investigate-zh':('Invents a diagnosis and TypeScript code instead of rewriting the investigation request; truncated.',['answered_request','truncated']),
'dev-investigate-en':('Invents a null-property diagnosis without source evidence; no-submit clause alone is insufficient.',['answered_request','invented_diagnosis']),
'dev-question-zh':('Answers in English with follow-up questions rather than relaying the Chinese question; truncated.',['answered_question','language_changed','truncated']),
'dev-risk-en':('Translates to Chinese rather than preserving message language; tentative cause and no-clear restriction remain.',['language_changed']),
'dev-review-zh':('Adds a request to provide feedback and an invented closing, expanding the original review/contact request.',['added_requirement','invented_closing']),
'dev-delay-zh':('Adds a subject, wrapper, unsupported courtesy and an assistant offer to revise the message.',['protocol_content','invented_closing']),
'dev-payment-en':('Translates to Chinese and introduces environment-recovery framing not stated by the source.',['language_changed','unsupported_recovery_framing']),
 'test-upload-zh':('Changes the requested file-size validation into an already-added completion claim.',['invented_completion']),
 'test-retry-zh':('Adds an instruction to start executing sendInvoice and invented procedural output.',['answered_request','added_requirement']),
 'test-retry-en':('Adds implementation notes and an explicit loop requirement, beyond rewriting the request.',['added_requirement','answered_request']),
 'test-upload-en':('Changes the request into completed work and adds API/no-fix assertions.',['invented_completion','added_requirement']),
 'test-request-en':('Changes the recipient-directed question into a speaker commitment to investigate.',['question_changed_to_commitment']),
 'test-incident-en':('Translates to Chinese, despite keeping the owner/action relationships.',['language_changed']),
 'test-migration-zh':('Adds speculative SQL/schema explanations instead of rewriting the limited investigation request; truncated.',['answered_request','truncated']),
 'test-migration-en':('Adds procedural next steps and reframes state preservation as before migrations rather than preserving current state.',['added_requirement']),
 'test-request-zh':('Changes a request for the recipient to inspect into a statement by the speaker that they have seen the cause.',['actor_or_intent_changed']),
 'test-estimate-en':('Translates to Chinese while preserving tentative cause and no-restart instruction.',['language_changed']),
 'test-availability-zh':('Drops the spoken apology, replacing it with requests for understanding and thanks.',['apology_dropped','invented_courtesy']),
 'test-export-en':('Translates the message to Chinese, with awkward ownership wording.',['language_changed'])}
scene_notes={
'dev-review-en':'Salutation, attachment status, Tuesday deadline, contact request and thanks retained in natural paragraphs.',
'dev-delay-en':'No-arrival status, expected Friday arrival and testing only after arrival all retained.',
'dev-uncertain-en':'Tentative six arrival and no-food-order request preserved.',
'dev-leave-zh':'Leave date/duration, handoff, approval request and thanks retained; salutation/body remain inline.',
'dev-owners-zh':'Both owners/tasks and exact dates retained, but a heading does not group each owner with their task/date.',
'dev-repair-en':'Thursday afternoon supersedes Wednesday; usual location retained, but oral correction remains.',
'dev-repair-zh':'Thursday afternoon correction and usual location retained, but stale Wednesday/correction phrasing remains.',
'dev-leave-en':'Thursday leave, Priya handoff and approval request retained with natural salutation/body.',
'dev-uncertain-zh':'Approximate six arrival, uncertainty and no-food-order request preserved.',
 'test-reschedule-en':'Monday-to-Tuesday move and three PM equivalence, same link and confirmation request all retained.',
 'test-availability-en':'Repairs Friday.s punctuation; tentative absence, uncertainty, future notice and apology retained.',
 'test-condition-en':'Both rain/online and no-rain/downstairs branches retained.',
 'test-invoice-zh':'Sent invoice, AB123 tracking, check/contact requests and thanks retained.',
 'test-export-zh':'Completed feature, pending security review, approval-before-release and Zheng ownership clearly separated.',
 'test-incident-zh':'API restoration, ongoing backfill, He/Song responsibilities and tomorrow-morning notice retained in clear sections.',
 'test-estimate-zh':'Uncertain connection-pool cause, monitor-first and no-restart intent retained in concise sections.',
 'test-correction-en':'Nine-hundred final budget, rejection of eight hundred and four people retained.',
 'test-correction-zh':'Resolves correction to nine hundred and preserves four people/no-mix-up request.',
 'test-reschedule-zh':'Monday-to-Tuesday three-PM move, unchanged link and confirmation request retained.',
 'test-invoice-en':'Sent status, AB123, check/contact requests and thanks retained.',
 'test-condition-zh':'Both rain/online and no-rain/downstairs branches retained.'}
for phase in ['dev','test']:
 out=OUT/phase;out.mkdir();records=[];reviews=[];sources=[]
 for arm in ['production','scene']:
  path=OLD/phase/(arm+'.jsonl');sources.append(dict(path=str(path.relative_to(ROOT)),sha256=sha(path)))
  for original in [json.loads(x) for x in path.read_text().splitlines()]:
   sample=samples[original['id']];response=original['response'];assert response['cleanup']['text']==sample['source']
   row=dict(original,variant=arm,source=sample['source'],scene=sample['scene'],language=sample['language'],context=sample['context'],cohort='retrospective-'+phase,originalPath=str(path.relative_to(ROOT)),originalRowSha256=rh(original))
   faithful=True;issues=[];useful=sample['scene'] in ['email','chat'];note='All inspected source facts, dates, roles, restrictions and request intent retained.'
   if arm=='production':
    if row['id']=='dev-investigate-zh':faithful=False;issues=['answered_request','truncated'];note='Invents an explanation of TS2345 instead of preserving the request; truncated.'
    if sample['scene']=='work':useful='risk-' in row['id'] or 'estimate-' in row['id']
    if row['id']=='dev-leave-en':useful=False;issues+=['overfragmented_body'];note+=' Each body sentence is a separate paragraph.'
    if row['id']=='dev-repair-en':useful=False;issues+=['oral_correction_retained'];note+=' Spoken Wednesday/actually-Thursday correction remains.'
    if sample['scene']=='prompt':useful=False;issues+=['missing_structure']
   else:
    if row['id'] in scene_bad:faithful=False;note,issues=scene_bad[row['id']]
    else:note=scene_notes[row['id']]
    useful=faithful and row['id'] not in ['dev-leave-zh','dev-owners-zh','dev-repair-en','dev-repair-zh']
    if faithful and not useful:issues+=['missing_natural_grouping']
   if not faithful:useful=False
   cleanup_useful=sample['scene']=='chat' or sample['scene']=='work' and ('risk-' in row['id'] or 'estimate-' in row['id'])
   reviews.append(dict(id=row['id'],variant=arm,rowSha256=rh(row),proposalFaithful=faithful,proposalSceneUseful=useful,cleanupFaithful=True,cleanupSceneUseful=cleanup_useful,repairSucceeded=False,issues=issues,note=note,cleanupNote='Cleanup equals inspected source exactly. All meaning retained; original layout and oral correction remain.'))
   records.append(row)
 (out/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in records))
 (out/'review.json').write_text(json.dumps(dict(reviewer='assistant-retrospective-manual-inspection',resultsSha256=sha(out/'results.jsonl'),reviews=reviews),ensure_ascii=False,indent=2)+'\n')
 (out/'manifest.json').write_text(json.dumps(dict(status='complete',kind='derived-view-of-original-inference',newGenerations=0,originalGenerations=len(records),resultsSha256=sha(out/'results.jsonl'),sources=sources,corpusSha256=sha(ROOT/'benchmarks/reproduction/scene-rewrite-alignment-v1/corpus.json'),scope='Post-hoc assistant inspection; original run, proxy report and selection are unchanged. Not independent review or new inference.'),indent=2)+'\n')
print('Reviewed and bound 96 historical generations; no new model calls.')
