"""Augment only selector data; preserve rules, examples, choices and formatter."""
import pathlib,importlib.util
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
prior=module('preview_prior',HERE.parent/'rewrite-selector-semantic-v2/pipeline.py')
preview=module('candidate_replacement_preview',HERE/'preview.py')
original_data=prior.selector.core.selection_data
prior.selector.core.HERE=HERE
prior.selector.core.selection_data=lambda prepared:preview.augment(original_data,prepared)
candidates_for=prior.candidates_for
selection_payload=prior.selection_payload
format_payload=prior.format_payload
def template_identity(arm,stage,scene):
 candidates_for(arm)
 if stage=='selection':return dict(version='selector-preview-v1-selection',path='benchmarks/reproduction/rewrite-selector-preview-v1/prompts.json')
 return prior.template_identity(arm,stage,scene)
