"""Export immutable expected payloads from the selected experimental producers."""
import importlib.util,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('selected_work_v4',ROOT/'benchmarks/reproduction/rewrite-work-scope-v4/pipeline.py')
pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
selectors=[]
for line in (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines():
 row=json.loads(line)
 if row['arm']=='baseline':selectors.append(dict(prepared=row['prepared'],request=row['request']))
formatters=[]
for scene in ['work','email','prompt','chat']:
 for source in ['小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。','Please confirm the schedule, do not send it yet.','给 fetchState 加校验，不改返回字段，也别动数据库。','……']:
  request,_=pipeline.format_payload('candidate',dict(scene=scene),source,dict(hypotheses=[]),{},[])
  formatters.append(dict(source=source,scene=scene,request=request))
(HERE/'payload-parity.json').write_text(json.dumps(dict(selectors=selectors,formatters=formatters),ensure_ascii=False,indent=2)+'\n')
