import json,pathlib,hashlib
r=pathlib.Path('.workspace/research/doubao-alignment')
items=[
('en','first-person-1','I I can send the invoice tonight.','I can send the invoice tonight.',['I can','invoice','tonight'],['I I','II']),
('en','first-person-2','I I I will not overwrite the archive.','I will not overwrite the archive.',['I will not','archive'],['I I','II']),
('en','quoted-pronoun','Leave "I I" in the transcript, do not normalize it.','Leave "I I" in the transcript, do not normalize it.',['"I I"','not normalize'],['"II"']),
('en','acronyms','Use P R for code review and A P I for the endpoint.','Use PR for code review and API for the endpoint.',['PR','API','endpoint'],[]),
('en','numeral','Section II must stay Section II.','Section II must stay Section II.',['Section II'],[]),
('en','tense','I had had enough, but I might return.','I had had enough, but I might return.',['had had','might'],[]),
('zh','repair','周一下午见面，啊不对，周三下午见面。','周三下午见面。',['周三','下午','见面'],['周一','不对']),
('zh','quoted-repeat','保留“我我”和“那个”，不要合并。','保留“我我”和“那个”，不要合并。',['“我我”','“那个”','不要'],[]),
('zh','reference','用那个，不是这个，别替我做决定。','用那个，不是这个，别替我做决定。',['那个','这个','别'],[]),
('zh','hedge','这个结论可能不成立，先别删除证据。','这个结论可能不成立，先别删除证据。',['可能','别','证据'],[]),
('zh','question','呃，那个，我想了解一下，升级何时完成？','我想了解一下，升级何时完成？',['升级','何时','完成','？'],['呃','那个']),
('zh','value','启动时间是九点，不对，十一点，别设闹钟。','启动时间是十一点，别设闹钟。',['十一点','别设闹钟'],['九点','不对']),
('en','repair','Keep the orange label, I mean the silver label.','Keep the silver label.',['silver label'],['orange','I mean']),
('en','parenthetical','I, like, need the export by Sunday.','I need the export by Sunday.',['export','by Sunday'],['like']),
('en','code','Keep `I I` and `P R` as literal code.','Keep `I I` and `P R` as literal code.',['`I I`','`P R`'],['`II`','`PR`']),
('en','comparison','That one is like the original, so keep it.','That one is like the original, so keep it.',['That one','like','original'],[]),
]
samples=[dict(id=f'{lang}-fresh-{tag}',language=lang,sourceText=src,references=[ref],requiredStrings=req,forbiddenStrings=no,protected=[],scenario='chat',mode='clean',approvedMemory=[],tags=[tag]) for lang,tag,src,ref,req,no in items]
p=r/'fresh-test.json'
with p.open('x') as f:json.dump({'version':1,'split':'test','annotation':'developer_authored_unreviewed','scope':'New post-review text fixtures, frozen before any model outputs on these cases','samples':samples},f,ensure_ascii=False,indent=2)
with (r/'fresh-freeze.json').open('x') as f:json.dump({'corpusSha256':hashlib.sha256(p.read_bytes()).hexdigest(),'caseCount':len(samples)},f)
p=r/'run-text.py';s=p.read_text().replace("corpora={'dev':", "corpora={'fresh':r/'fresh-test.json','dev':");p.write_text(s)
