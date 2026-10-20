"""Reuse source-script routing and encoding with a versioned formatting recipe."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('history_v2_routed_formatter',HERE.parent/'rewrite-language-profile-v2/pipeline.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
prior.read=lambda:json.loads((HERE/'prompts.json').read_text())
def format_payload(sample,edited,prepared,choices,edits):
 request,encoding=prior.format_payload(sample,edited,prepared,choices,edits);request['cache_prompt']=True
 return request,encoding
