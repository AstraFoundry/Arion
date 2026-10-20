"""Reuse script-routed formatting; change only the candidate work recipe."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
baseline=module('work_v3_baseline_formatter',HERE.parent/'rewrite-history-format-v2/pipeline.py')
candidate=module('work_v3_candidate_formatter',HERE.parent/'rewrite-language-profile-v2/pipeline.py');candidate.read=lambda:json.loads((HERE/'prompts.json').read_text())
def format_payload(arm,sample,edited,prepared,choices,edits):
 target=baseline if arm=='baseline' else candidate if arm=='candidate' else None
 if target is None:raise ValueError('Unknown work formatter arm')
 request,encoding=target.format_payload(sample,edited,prepared,choices,edits);request['cache_prompt']=True
 return request,encoding
