import json,pathlib
base=pathlib.Path('.workspace/correction-chain-expanded-v2')
decisions=json.loads((base/'baseline-review.json').read_text())
zh=[
('U0 U0 U0',None),('D0 D0 D0',None),('P0 P0 P0',None),('D0 D0 P1','Offline retains parcel shipping to Hangzhou with a minor 进/寄 spelling artifact.'),
('P0 P0 P0',None),('D0 D0 P0',None),
('P0 U0 P0','Direct rewriting leaves an explicit spoken cancellation repair. Correction plus rewriting removes 算了 and leaves conflicting cancel-training and training-as-usual assertions, so the final status is ambiguous.'),
('P0 P0 U0',None),('P2 P2 P2',None),('D1 D1 D1',None),('P2 P2 P2',None),('D1 D1 P2',None),
('P2 P2 P2',None),('D1 D1 P2',None),('P2 P2 P2',None),('D1 D1 D1',None),
('P1 P1 P1','The process-simplification question is preserved; initial 就是 and small punctuation issues are minor readable residue.'),
('D1 D1 D0','Mosquito nets replace documents; the correction chain additionally introduces a second net object, while offline repeats the corrupted object.'),
('P2 P2 P2',None),
('D0 D0 P1','Direct and corrected rewrites retain the unsupported 来人/赖人 fragment; offline preserves confirming the responsible person with only minor 那个 residue.'),
('P2 P2 P2',None),('D1 D1 P2',None),('P2 P2 P2',None),('D1 D1 U1',None),
('P2 P2 P2',None),('P2 P2 P2',None),('P2 P2 P2',None),('D1 D1 D1',None),('P2 P2 P2',None),
('D1 D1 D2','The delivered status is still equipment not manufactured rather than not arrived; attractive offline bullet structure does not repair this material fact.'),
('P2 P2 P2',None),('D1 D1 D1',None)]
en=[
('P2 D2 P2',None),
('P0 D2 P0','Direct and offline rewrites retain the unresolved recipient repair; the correction-chain rewrite explicitly says send to Alice and do not send to Brian, reversing the final recipient.'),
('P2 D2 P2',None),
('D2 P0 P2','Direct rewrite selects Shiv as the final action instead of ship; the correction chain retains I mean before the Seattle shipping action; offline correctly ships the package to Seattle.'),
('P2 D2 P2',None),('P0 D2 P0',None),
('D2 P0 D2','Direct and offline rewriting remove the second paragraph and preserve the first, contradicting the final keep-second instruction; the correction chain retains the unresolved but recoverable repair.'),
('P0 D2 P0',None),('P2 P2 P2',None),('P2 P2 P2',None),('P2 D2 P2',None),
('P2 P2 P2',None),('P2 P2 P2',None),('P2 D2 P2',None),('P2 P2 P2',None),('P2 P2 P2',None),
('P1 P1 P2','The tentative smaller-venue proposal is intact; direct ellipsis and corrected oh are minor conversational residue, while offline prose is clean.'),
('P0 P2 P0','Direct and offline retain you know; the correction chain delivers a clean inventory-check statement.'),
('P0 P0 P0',None),('P2 P2 P2',None),('P2 P2 P2',None),('P1 P1 U1',None),
('P2 P2 P2',None),('P2 P2 P2',None),('P2 P2 P2',None),('P2 P2 P2',None),('P2 P2 P2',None),
('P1 P1 P2',None),('P2 D1 P2',None),('P2 D2 P2',None),('P2 D2 P2',None),('P2 D2 P2',None)]
mixed=[
('P2 P2 P2',None),('D1 D1 P0',None),('U1 U1 U1',None),
('D2 D2 D1','Final recipient David is absent; direct and correction rewrite choose Alice, and offline explicitly rejects another incorrect recipient.'),
('D1 D1 P1','Streaming rewrites retain incorrect ATI; offline preserves final fifteen seconds and API path with a minor 射程/设成 spelling artifact.'),
('D0 D0 D0',None),('P2 P1 P2','GPU memory remains the final target; corrected wording has a minor 地/的 grammatical artifact.'),
('D0 D0 D0',None),('D1 D1 P2',None),('D1 D1 D1',None),('D1 D1 P2',None),('D1 D1 D1',None),
('P2 P2 P2',None),('D1 D1 D1',None),('D1 D1 P2',None),('D1 D1 U1',None),('P2 P2 P2',None),
('D1 D1 D0','README is replaced with a case to update; offline additionally retains fillers and repetition.'),
('P2 P2 P2',None),('D1 D1 D1',None),('P2 P1 P2',None),('D1 D1 D1',None),
('D1 D1 P1',None),('D1 D1 U1',None),('P2 P2 P2',None),('D1 D1 D1',None),('P2 P2 P2',None),
('D1 D1 U1',None),('P2 P2 P2',None),('D1 D1 D1',None),('P1 P1 P1',None),
('D1 D1 U1','Streaming loses export permission and changes synchronization to rerecording. Offline restores CSV permission but asks about uploading again, presupposing an earlier upload absent from the script; that history remains uncertain.')]
for lang,rows in [('zh',zh),('en',en),('mixed',mixed)]:
 assert len(rows)==32
 for i,(codes,note) in enumerate(rows):
  case=f'expanded-{lang}-{i:02d}'
  d=decisions[case];a,c,o=d.pop('baselineCodes');l,cl,ol=codes.split()
  reason=d.pop('baselineReason');d['codes']=[a,c,l,cl,o,ol];d['reasons']={}
  for arm,grade in zip(['asr','asr_correct','offline'],[a,c,o]):
   if grade!='P2':d['reasons'][arm]=reason
  for arm,grade in zip(['asr_llm','asr_correct_llm','offline_llm'],[l,cl,ol]):
   if grade!='P2':d['reasons'][arm]=note or reason
for i in range(4):decisions[f'expanded-silence-{i}']={'codes':['P2']*6,'reasons':{}}
(base/'decisions.json').write_text(json.dumps(decisions,ensure_ascii=False,indent=2)+'\n')
