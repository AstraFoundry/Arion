"""Retain complete current inputs and trusted rules while removing demonstrations."""
import importlib.util,json
from pathlib import Path
HERE=Path(__file__).resolve().parent;ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('d1_original_structured_layout',HERE.parent/'rewrite-selector-d1-layout-v1/layout.py');base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
def convert(original,kind,prepared):
 value=base.convert(original,kind,prepared);value['state'].pop('demonstrations')
 for question in value['questions'].values():
  if kind=='structured':question['instructions']=question['instructions'].replace('demonstrations 只示范拼写选择；','')
  elif kind=='sentence':question['instructions']=question['instructions'].replace('demonstrations 仅为示例，source 才是本次口述。','source 是本次口述。')
 return value
def validate(records,kind):
 references={(r['arm'],r['id']):r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};seen=set()
 for item in records:
  row=references['baseline',item['id']]
  if item['id'] in seen or item['request']!=convert(row['request'],kind,row['prepared']) or item['scene']!=row['scene']:raise ValueError('Changed compact native payload or complete current input')
  seen.add(item['id'])
 if len(records)!=6:raise ValueError('Incomplete compact input scope')
 return records
