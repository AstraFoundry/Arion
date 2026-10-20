"""Reuse routed work formatting with a shorter contrastive candidate recipe."""
import pathlib,importlib.util,json
HERE=pathlib.Path(__file__).resolve().parent
s=importlib.util.spec_from_file_location('work_v4_formatter',HERE.parent/'rewrite-work-scope-v3/pipeline.py');prior=importlib.util.module_from_spec(s);s.loader.exec_module(prior)
prior.candidate.read=lambda:json.loads((HERE/'prompts.json').read_text())
format_payload=prior.format_payload
